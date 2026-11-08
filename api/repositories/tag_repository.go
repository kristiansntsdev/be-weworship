package repositories

import (
	"database/sql"
	"strings"

	"be-songbanks-v1/api/models"
	"github.com/jmoiron/sqlx"
)

type TagRepository struct {
	db *sqlx.DB
}

func NewTagRepository(db *sqlx.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) List(search string) ([]models.Tag, error) {
	search = strings.TrimSpace(search)
	query := `SELECT id,name,description FROM tags`
	args := []any{}
	if search != "" {
		query += ` WHERE name ILIKE ? ESCAPE '\' OR description ILIKE ? ESCAPE '\'`
		like := containsPattern(search)
		args = append(args, like, like)
	}
	query = r.db.Rebind(query + ` ORDER BY name ASC`)
	rows := []models.Tag{}
	err := r.db.Select(&rows, query, args...)
	return rows, err
}

func (r *TagRepository) FindByName(name string) (*models.Tag, error) {
	var t models.Tag
	err := r.db.Get(&t, r.db.Rebind(`SELECT id,name,description FROM tags WHERE name=? LIMIT 1`), name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TagRepository) Create(name string) (int, error) {
	var id int
	err := r.db.QueryRow(r.db.Rebind(`INSERT INTO tags (name,"createdAt","updatedAt") VALUES (?,NOW(),NOW()) RETURNING id`), name).Scan(&id)
	return id, err
}

func (r *TagRepository) GetTagsForSong(songID int) ([]models.Tag, error) {
	rows := []models.Tag{}
	err := r.db.Select(&rows, r.db.Rebind(`SELECT t.id,t.name,t.description FROM tags t JOIN song_tags st ON t.id=st.tag_id WHERE st.song_id=?`), songID)
	return rows, err
}

// CountPlayableSongsByTagNames counts playable songs (ChordPro, not DMCA'd) per tag, keyed by lower-cased tag name.
// Tags with no playable songs are absent.
func (r *TagRepository) CountPlayableSongsByTagNames(names []string) (map[string]int, error) {
	out := map[string]int{}
	if len(names) == 0 {
		return out, nil
	}
	lower := make([]string, len(names))
	for i, n := range names {
		lower[i] = strings.ToLower(n)
	}
	rows := []struct {
		Name  string `db:"name"`
		Count int    `db:"count"`
	}{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT LOWER(t.name) AS name, COUNT(*)::int AS count
		FROM tags t
		JOIN song_tags st ON st.tag_id = t.id
		JOIN songs s ON s.id = st.song_id
		WHERE LOWER(t.name) = ANY(?)
		  AND s.lyrics_and_chords LIKE '%[%' AND s.lyrics_and_chords NOT LIKE '%<span%'
		  AND s.dmca_takedown = false
		GROUP BY LOWER(t.name)`), lower)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Name] = row.Count
	}
	return out, nil
}

// GetTagsForSongs loads the tags of several songs in one query, keyed by song id. Songs without tags are absent.
func (r *TagRepository) GetTagsForSongs(songIDs []int) (map[int][]models.Tag, error) {
	bySong := map[int][]models.Tag{}
	if len(songIDs) == 0 {
		return bySong, nil
	}
	ids64 := make([]int64, len(songIDs))
	for i, id := range songIDs {
		ids64[i] = int64(id)
	}
	rows := []struct {
		SongID int `db:"song_id"`
		models.Tag
	}{}
	err := r.db.Select(&rows, r.db.Rebind(`SELECT st.song_id,t.id,t.name,t.description FROM tags t JOIN song_tags st ON t.id=st.tag_id WHERE st.song_id = ANY(?)`), ids64)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		bySong[row.SongID] = append(bySong[row.SongID], row.Tag)
	}
	return bySong, nil
}
