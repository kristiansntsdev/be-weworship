package repositories

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

type EmbeddingRepository struct {
	db *sqlx.DB
}

func NewEmbeddingRepository(db *sqlx.DB) *EmbeddingRepository {
	return &EmbeddingRepository{db: db}
}

// SongChunk is one embedded piece of a song (index 0 = whole song, 1.. = stanzas).
type SongChunk struct {
	Index  int
	Text   string
	Vector []float32
}

// SongDistance is a song's closest chunk distance to a query (cosine distance: 0 = identical, 2 = opposite).
type SongDistance struct {
	SongID   int     `db:"song_id"`
	Distance float64 `db:"distance"`
}

// EmbeddingSource is the song text that embeddings are built from.
type EmbeddingSource struct {
	ID             int            `db:"id"`
	Title          string         `db:"title"`
	Artist         sql.NullString `db:"artist"`
	PlainLyrics    sql.NullString `db:"plain_lyrics"`
	LyricsAndChord sql.NullString `db:"lyrics_and_chords"`
}

// formatVector renders a vector as a pgvector literal, e.g. "[0.1,-0.2]".
func formatVector(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// parseVector reads a pgvector text literal ("[0.1,-0.2]") back into a vector — the inverse of formatVector.
func parseVector(s string) ([]float32, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, fmt.Errorf("invalid vector literal")
	}
	body := s[1 : len(s)-1]
	if strings.TrimSpace(body) == "" {
		return []float32{}, nil
	}
	parts := strings.Split(body, ",")
	out := make([]float32, len(parts))
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid vector literal: %w", err)
		}
		out[i] = float32(f)
	}
	return out, nil
}

// MixCandidate is a playable song near a taste or mood vector, with what a mix needs to list it.
type MixCandidate struct {
	SongID   int            `db:"song_id"`
	Distance float64        `db:"distance"`
	Title    string         `db:"title"`
	Artist   sql.NullString `db:"artist"` // JSON array, see utils.ParseArtists
	Key      sql.NullString `db:"base_chord"`
}

// SongVectors returns each song's whole-song embedding (chunk 0). Songs without one are left out.
func (r *EmbeddingRepository) SongVectors(songIDs []int, model string) (map[int][]float32, error) {
	out := map[int][]float32{}
	if len(songIDs) == 0 {
		return out, nil
	}
	ids64 := make([]int64, len(songIDs))
	for i, id := range songIDs {
		ids64[i] = int64(id)
	}
	rows := []struct {
		SongID    int    `db:"song_id"`
		Embedding string `db:"embedding"`
	}{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT song_id, embedding::text AS embedding
		FROM song_embeddings
		WHERE model = ? AND chunk_index = 0 AND song_id = ANY(?)`), model, ids64)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		vec, err := parseVector(row.Embedding)
		if err != nil {
			return nil, fmt.Errorf("song %d: %w", row.SongID, err)
		}
		out[row.SongID] = vec
	}
	return out, nil
}

// NearestPlayableSongs ranks playable songs (ChordPro, not DMCA'd — same rule as SongRepository.ListRecent) by how
// close their whole-song embedding (chunk 0) is to the vector, nearest first. Unlike NearestSongs it ignores stanza
// chunks: a mix wants songs that are alike as a whole, not songs sharing one line.
func (r *EmbeddingRepository) NearestPlayableSongs(vector []float32, model string, limit int) ([]MixCandidate, error) {
	rows := []MixCandidate{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT e.song_id, e.embedding <=> ?::vector AS distance, s.title, s.artist, s.base_chord
		FROM song_embeddings e
		JOIN songs s ON s.id = e.song_id
		WHERE e.model = ? AND e.chunk_index = 0
		  AND s.lyrics_and_chords LIKE '%[%' AND s.lyrics_and_chords NOT LIKE '%<span%'
		  AND s.dmca_takedown = false
		ORDER BY distance ASC
		LIMIT ?`), formatVector(vector), model, limit)
	return rows, err
}

// ReplaceSongChunks swaps all of a song's chunks atomically.
func (r *EmbeddingRepository) ReplaceSongChunks(songID int, model, hash string, chunks []SongChunk) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op once committed

	if _, err := tx.Exec(tx.Rebind(`DELETE FROM song_embeddings WHERE song_id=?`), songID); err != nil {
		return err
	}
	if len(chunks) > 0 {
		rows := make([]string, len(chunks))
		args := make([]any, 0, len(chunks)*6)
		for i, c := range chunks {
			rows[i] = "(?,?,?,?::vector,?,?)"
			args = append(args, songID, c.Index, c.Text, formatVector(c.Vector), model, hash)
		}
		q := `INSERT INTO song_embeddings (song_id, chunk_index, chunk_text, embedding, model, content_hash) VALUES ` + strings.Join(rows, ",")
		if _, err := tx.Exec(tx.Rebind(q), args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// StoredHashes maps song_id → content_hash for songs already embedded with the given model.
func (r *EmbeddingRepository) StoredHashes(model string) (map[int]string, error) {
	rows := []struct {
		SongID int    `db:"song_id"`
		Hash   string `db:"content_hash"`
	}{}
	if err := r.db.Select(&rows, r.db.Rebind(`SELECT DISTINCT song_id, content_hash FROM song_embeddings WHERE model=?`), model); err != nil {
		return nil, err
	}
	out := make(map[int]string, len(rows))
	for _, row := range rows {
		out[row.SongID] = row.Hash
	}
	return out, nil
}

// NearestSongs returns songs whose closest chunk is within maxDistance of the query vector, nearest first.
// Exact scan (no ANN index) — see migrations/add_song_embeddings.sql.
func (r *EmbeddingRepository) NearestSongs(vector []float32, model string, maxDistance float64, limit int) ([]SongDistance, error) {
	rows := []SongDistance{}
	err := r.db.Select(&rows, r.db.Rebind(`
		WITH q AS (SELECT ?::vector AS v)
		SELECT e.song_id, MIN(e.embedding <=> q.v) AS distance
		FROM song_embeddings e, q
		WHERE e.model = ?
		GROUP BY e.song_id
		HAVING MIN(e.embedding <=> q.v) <= ?
		ORDER BY distance ASC
		LIMIT ?`), formatVector(vector), model, maxDistance, limit)
	return rows, err
}

// Source returns the text of one song for embedding; nil when the song does not exist.
func (r *EmbeddingRepository) Source(songID int) (*EmbeddingSource, error) {
	var row EmbeddingSource
	err := r.db.Get(&row, r.db.Rebind(`SELECT id, title, artist, plain_lyrics, lyrics_and_chords FROM songs WHERE id=?`), songID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// AllSources returns the text of every song, for the backfill.
func (r *EmbeddingRepository) AllSources() ([]EmbeddingSource, error) {
	rows := []EmbeddingSource{}
	err := r.db.Select(&rows, `SELECT id, title, artist, plain_lyrics, lyrics_and_chords FROM songs ORDER BY id`)
	return rows, err
}
