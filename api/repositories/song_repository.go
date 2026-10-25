package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/utils"
	"github.com/jmoiron/sqlx"
)

type SongRepository struct {
	db *sqlx.DB
}

func NewSongRepository(db *sqlx.DB) *SongRepository {
	return &SongRepository{db: db}
}

// songWordsClause requires every search word to appear in the title, artist, or lyrics
// (word by word, so HTML/chord markup between words does not break a match). Empty search → "".
func songWordsClause(search string) (string, []any) {
	words := strings.Fields(strings.TrimSpace(search))
	if len(words) == 0 {
		return "", nil
	}
	conds := make([]string, len(words))
	args := make([]any, 0, len(words)*3)
	for i, word := range words {
		conds[i] = `(s.title ILIKE ? ESCAPE '\' OR s.artist ILIKE ? ESCAPE '\' OR COALESCE(s.plain_lyrics, s.lyrics_and_chords) ILIKE ? ESCAPE '\')`
		like := containsPattern(word)
		args = append(args, like, like, like)
	}
	return "(" + strings.Join(conds, " AND ") + ")", args
}

func (r *SongRepository) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, keys, artists []string, hasLink, chordPro *bool) ([]models.Song, int, error) {
	offset := (page - 1) * limit
	where := []string{"1=1"}
	args := []any{}

	if clause, wordArgs := songWordsClause(search); clause != "" {
		where = append(where, clause)
		args = append(args, wordArgs...)
	}
	filters, filterArgs := songFilterClauses(baseChord, tagIDs, keys, artists, hasLink, chordPro)
	where = append(where, filters...)
	args = append(args, filterArgs...)

	whereClause := strings.Join(where, " AND ")
	countArgs := append([]any{}, args...)
	var total int
	countQ := r.db.Rebind(`SELECT COUNT(DISTINCT s.id) FROM songs s WHERE ` + whereClause)
	if err := r.db.Get(&total, countQ, countArgs...); err != nil {
		return nil, 0, err
	}

	query := r.db.Rebind(`SELECT s.id,s.slug,s.title,s.artist,s.base_chord,s.bpm,s.lyrics_and_chords,s.external_links,s.dmca_takedown,s.dmca_status_notes,s."createdAt",s."updatedAt" FROM songs s WHERE ` + whereClause + ` ORDER BY ` + sortBy + ` ` + sortOrder + `, s.title ASC, s.id ASC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)
	rows := []models.Song{}
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// SearchCandidate is a song matched by a ranked search, with its text-match score.
type SearchCandidate struct {
	ID       int     `db:"id"`
	Title    string  `db:"title"`
	LexScore float64 `db:"lex_score"`
	LexMatch bool    `db:"lex_match"`
	// Strength: 2 = exact title, 1 = a query of 3+ words found verbatim in the title or lyrics, 0 = neither.
	Strength int `db:"strength"`
}

// SearchCandidates returns every filtered song that matches the search text — all words present, or a close
// trigram match on the title or lyrics so typos still hit — plus extraIDs (semantic matches), each with a text
// score. Not paginated: the service merges this with the semantic ranking and pages the result.
func (r *SongRepository) SearchCandidates(search string, extraIDs []int, baseChord string, tagIDs []int, keys, artists []string, hasLink, chordPro *bool) ([]SearchCandidate, error) {
	q := strings.TrimSpace(search)
	wordsClause, wordArgs := songWordsClause(q)
	if wordsClause == "" {
		wordsClause = "FALSE"
	}
	lyrics := `COALESCE(s.plain_lyrics, s.lyrics_and_chords, '')`
	phrase := containsPattern(q)
	prefix := escapeLikePattern(q) + "%"

	// Exact title > title prefix > phrase in title/lyrics > all words present, plus trigram closeness for typos.
	score := `(CASE WHEN lower(s.title) = lower(?) THEN 4 ELSE 0 END)
		+ (CASE WHEN s.title ILIKE ? ESCAPE '\' THEN 2 ELSE 0 END)
		+ (CASE WHEN s.title ILIKE ? ESCAPE '\' THEN 1.5 ELSE 0 END)
		+ (CASE WHEN ` + lyrics + ` ILIKE ? ESCAPE '\' THEN 1.5 ELSE 0 END)
		+ (CASE WHEN s.artist ILIKE ? ESCAPE '\' THEN 1 ELSE 0 END)
		+ (CASE WHEN ` + wordsClause + ` THEN 1 ELSE 0 END)
		+ 1.5 * word_similarity(?, s.title)
		+ word_similarity(?, ` + lyrics + `)`
	// <% = pg_trgm word similarity above pg_trgm.word_similarity_threshold (default 0.6).
	match := `(` + wordsClause + ` OR ? <% s.title OR ? <% ` + lyrics + `)`
	strength := `(CASE WHEN lower(s.title) = lower(?) THEN 2
		WHEN ? AND (s.title ILIKE ? ESCAPE '\' OR ` + lyrics + ` ILIKE ? ESCAPE '\') THEN 1 ELSE 0 END)`
	phraseQuery := len(strings.Fields(q)) >= 3

	args := []any{q, prefix, phrase, phrase, phrase}
	args = append(args, wordArgs...)
	args = append(args, q, q)
	args = append(args, wordArgs...)
	args = append(args, q, q)
	args = append(args, q, phraseQuery, phrase, phrase)

	where := "TRUE"
	filters, filterArgs := songFilterClauses(baseChord, tagIDs, keys, artists, hasLink, chordPro)
	if len(filters) > 0 {
		where = strings.Join(filters, " AND ")
	}
	args = append(args, filterArgs...)

	ids := make([]int64, len(extraIDs))
	for i, id := range extraIDs {
		ids[i] = int64(id)
	}
	args = append(args, ids)

	query := `SELECT id, title, lex_score, lex_match, strength FROM (
		SELECT s.id, s.title, (` + score + `)::float8 AS lex_score, ` + match + ` AS lex_match, ` + strength + ` AS strength
		FROM songs s WHERE ` + where + `
	) c WHERE c.lex_match OR c.id = ANY(?)`
	rows := []SearchCandidate{}
	err := r.db.Select(&rows, r.db.Rebind(query), args...)
	return rows, err
}

// ListByIDs loads full song rows for the given ids (unordered).
func (r *SongRepository) ListByIDs(ids []int) ([]models.Song, error) {
	rows := []models.Song{}
	if len(ids) == 0 {
		return rows, nil
	}
	ids64 := make([]int64, len(ids))
	for i, id := range ids {
		ids64[i] = int64(id)
	}
	err := r.db.Select(&rows, r.db.Rebind(`SELECT s.id,s.slug,s.title,s.artist,s.base_chord,s.bpm,s.lyrics_and_chords,s.external_links,s.dmca_takedown,s.dmca_status_notes,s."createdAt",s."updatedAt" FROM songs s WHERE s.id = ANY(?)`), ids64)
	return rows, err
}

// songFilterClauses builds the non-search filters shared by List and SearchCandidates.
func songFilterClauses(baseChord string, tagIDs []int, keys, artists []string, hasLink, chordPro *bool) ([]string, []any) {
	where := []string{}
	args := []any{}
	if strings.TrimSpace(baseChord) != "" {
		where = append(where, `s.base_chord ILIKE ? ESCAPE '\'`)
		args = append(args, containsPattern(baseChord))
	}
	if len(keys) > 0 {
		// Exact match on any of the given keys (unlike base_chord, where "A" would also match "Am" and "Ab").
		ph := strings.TrimRight(strings.Repeat("?,", len(keys)), ",")
		where = append(where, `s.base_chord IN (`+ph+`)`)
		for _, k := range keys {
			args = append(args, k)
		}
	}
	if len(artists) > 0 {
		// artist holds a JSON array. Rows written by Go's json.Marshal store &, <, > as unicode escapes while
		// imported rows keep them literal, so match the quoted element in both encodings.
		conds := make([]string, len(artists))
		for i, a := range artists {
			conds[i] = `(s.artist ILIKE ? ESCAPE '\' OR s.artist ILIKE ? ESCAPE '\')`
			escaped, _ := json.Marshal(a)
			args = append(args, containsPattern(`"`+a+`"`), containsPattern(string(escaped)))
		}
		where = append(where, "("+strings.Join(conds, " OR ")+")")
	}
	if len(tagIDs) > 0 {
		ph := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
		where = append(where, `s.id IN (SELECT DISTINCT song_id FROM song_tags WHERE tag_id IN (`+ph+`))`)
		for _, id := range tagIDs {
			args = append(args, id)
		}
	}
	if hasLink != nil {
		if *hasLink {
			where = append(where, `(s.external_links IS NOT NULL AND s.external_links::text NOT IN ('', 'null', '{}'))`)
		} else {
			where = append(where, `(s.external_links IS NULL OR s.external_links::text IN ('', 'null', '{}'))`)
		}
	}
	if chordPro != nil {
		if *chordPro {
			where = append(where, `(s.lyrics_and_chords LIKE '%[%' AND s.lyrics_and_chords NOT LIKE '%<span%')`)
		} else {
			where = append(where, `(s.lyrics_and_chords IS NULL OR s.lyrics_and_chords NOT LIKE '%[%' OR s.lyrics_and_chords LIKE '%<span%')`)
		}
	}
	return where, args
}

func (r *SongRepository) GetByID(id int) (*models.Song, error) {
	var row models.Song
	err := r.db.Get(&row, r.db.Rebind(`SELECT id,slug,title,artist,base_chord,bpm,lyrics_and_chords,external_links,dmca_takedown,dmca_status_notes,"createdAt","updatedAt" FROM songs WHERE id=?`), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *SongRepository) GetBySlug(slug string) (*models.Song, error) {
	var row models.Song
	err := r.db.Get(&row, r.db.Rebind(`SELECT id,slug,title,artist,base_chord,bpm,lyrics_and_chords,external_links,dmca_takedown,dmca_status_notes,"createdAt","updatedAt" FROM songs WHERE slug=?`), slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *SongRepository) Create(title, artistJSON string, baseChord *string, bpm *int, lyrics *string, externalLinks *string, dmcaTakedown bool, dmcaStatusNotes *string, baseSlug string) (int, error) {
	// Extract plain lyrics from formatted lyrics
	var plainLyrics *string
	if lyrics != nil && *lyrics != "" {
		plain := utils.ExtractPlainLyrics(*lyrics)
		plainLyrics = &plain
	}

	var id int
	err := r.db.QueryRow(r.db.Rebind(`INSERT INTO songs (slug,title,artist,base_chord,bpm,lyrics_and_chords,plain_lyrics,external_links,dmca_takedown,dmca_status_notes,"createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?,?,?,NOW(),NOW()) RETURNING id`), baseSlug, title, artistJSON, baseChord, bpm, lyrics, plainLyrics, externalLinks, dmcaTakedown, dmcaStatusNotes).Scan(&id)
	if err != nil {
		return 0, err
	}
	// Finalize slug to "base-{id}" to guarantee uniqueness
	finalSlug := fmt.Sprintf("%s-%d", baseSlug, id)
	_, err = r.db.Exec(r.db.Rebind(`UPDATE songs SET slug=? WHERE id=?`), finalSlug, id)
	return id, err
}

func (r *SongRepository) UpdateFields(id int, setExpr string, args ...any) (int64, error) {
	query := r.db.Rebind(`UPDATE songs SET ` + setExpr + `,"updatedAt"=NOW() WHERE id=?`)
	args = append(args, id)
	res, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *SongRepository) DeleteByID(id int) (int64, error) {
	res, err := r.db.Exec(r.db.Rebind(`DELETE FROM songs WHERE id=?`), id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *SongRepository) ClearSongTags(songID int) error {
	_, err := r.db.Exec(r.db.Rebind(`DELETE FROM song_tags WHERE song_id=?`), songID)
	return err
}

func (r *SongRepository) AssignSongTag(songID, tagID int) error {
	_, err := r.db.Exec(r.db.Rebind(`INSERT INTO song_tags (song_id,tag_id,"createdAt","updatedAt") VALUES (?,?,NOW(),NOW()) ON CONFLICT DO NOTHING`), songID, tagID)
	return err
}

func (r *SongRepository) ExistsByID(id int) (bool, error) {
	var count int
	if err := r.db.Get(&count, r.db.Rebind(`SELECT COUNT(*) FROM songs WHERE id=?`), id); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SongRepository) ListArtistsRaw() ([]string, error) {
	rows := []string{}
	err := r.db.Select(&rows, `
		SELECT artist FROM songs
		WHERE artist IS NOT NULL AND artist != ''
		  AND (lyrics_and_chords LIKE '%[%' AND lyrics_and_chords NOT LIKE '%<span%')
		  AND dmca_takedown = false`)
	return rows, err
}

// KeyCount is one distinct base_chord and how many visible songs use it.
type KeyCount struct {
	Key   string `db:"song_key" json:"key"`
	Count int    `db:"song_count" json:"count"`
}

// ListKeyCounts returns the distinct base_chord values of ChordPro songs, for key filters.
func (r *SongRepository) ListKeyCounts() ([]KeyCount, error) {
	rows := []KeyCount{}
	err := r.db.Select(&rows, `
		SELECT base_chord AS song_key, COUNT(*) AS song_count FROM songs
		WHERE base_chord IS NOT NULL AND TRIM(base_chord) != ''
		  AND (lyrics_and_chords LIKE '%[%' AND lyrics_and_chords NOT LIKE '%<span%')
		  AND dmca_takedown = false
		GROUP BY base_chord
		ORDER BY base_chord`)
	return rows, err
}

// Count returns the number of songs with ChordPro content available.
func (r *SongRepository) Count() (int, error) {
	var n int
	err := r.db.Get(&n, `
		SELECT COUNT(*) FROM songs
		WHERE (lyrics_and_chords LIKE '%[%' AND lyrics_and_chords NOT LIKE '%<span%')
		  AND dmca_takedown = false`)
	return n, err
}

// ListRecent returns the newest playable (ChordPro, non-DMCA) songs by createdAt.
func (r *SongRepository) ListRecent(limit int) ([]models.Song, error) {
	if limit < 1 {
		limit = 8
	}
	rows := []models.Song{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT id, slug, title, artist, base_chord, bpm, lyrics_and_chords, external_links,
		       dmca_takedown, dmca_status_notes, "createdAt", "updatedAt"
		FROM songs
		WHERE (lyrics_and_chords LIKE '%[%' AND lyrics_and_chords NOT LIKE '%<span%')
		  AND dmca_takedown = false
		ORDER BY "createdAt" DESC, id DESC
		LIMIT ?`), limit)
	return rows, err
}

func (r *SongRepository) ListAllChordPro() ([]models.Song, error) {
	rows := []models.Song{}
	err := r.db.Select(&rows, `SELECT id,slug,title,artist,base_chord,bpm,lyrics_and_chords,external_links,dmca_takedown,dmca_status_notes,"createdAt","updatedAt" FROM songs WHERE (lyrics_and_chords LIKE '%[%' AND lyrics_and_chords NOT LIKE '%<span%') AND dmca_takedown = false ORDER BY title ASC`)
	return rows, err
}

// ── Song Requests ─────────────────────────────────────────────────────────────

func (r *SongRepository) CreateSongRequest(userID int, songTitle, referenceLink, lyricsType, lyrics string) (*models.SongRequest, error) {
	req := &models.SongRequest{}
	err := r.db.QueryRowx(
		`INSERT INTO song_requests (user_id, song_title, reference_link, lyrics_type, lyrics) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, song_title, reference_link, lyrics_type, lyrics, status, admin_notes, "createdAt", "updatedAt"`,
		userID, songTitle, referenceLink, lyricsType, lyrics,
	).StructScan(req)
	return req, err
}

func (r *SongRepository) ListSongRequests(status string, page, limit int) ([]models.SongRequest, int, error) {
	offset := (page - 1) * limit
	where := "1=1"
	args := []any{}

	if status != "" {
		where = "status = $1"
		args = append(args, status)
	}

	var total int
	countQ := `SELECT COUNT(*) FROM song_requests WHERE ` + where
	if err := r.db.QueryRowx(r.db.Rebind(countQ), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dbRows := []models.SongRequestRow{}
	args = append(args, limit, offset)
	q := r.db.Rebind(`SELECT id, user_id, song_title, reference_link, lyrics_type, lyrics, status, admin_notes, "createdAt", "updatedAt"
		FROM song_requests WHERE ` + where + ` ORDER BY "createdAt" DESC LIMIT ? OFFSET ?`)
	if err := r.db.Select(&dbRows, q, args...); err != nil {
		return nil, 0, err
	}

	rows := make([]models.SongRequest, len(dbRows))
	for i, dbRow := range dbRows {
		rows[i] = *dbRow.ToSongRequest()
	}
	return rows, total, nil
}

func (r *SongRepository) UpdateSongRequestStatus(id int, status, adminNotes string) error {
	_, err := r.db.Exec(
		`UPDATE song_requests SET status = $1, admin_notes = $2, "updatedAt" = NOW() WHERE id = $3`,
		status, adminNotes, id,
	)
	return err
}

func (r *SongRepository) ClearSongRequestLyrics(id int) error {
	_, err := r.db.Exec(
		`UPDATE song_requests SET lyrics = '', "updatedAt" = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *SongRepository) GetSongRequestByID(id int) (*models.SongRequest, bool, error) {
	dbRow := &models.SongRequestRow{}
	err := r.db.QueryRowx(
		`SELECT id, user_id, song_title, reference_link, lyrics_type, lyrics, status, admin_notes, "createdAt", "updatedAt" FROM song_requests WHERE id = $1`,
		id,
	).StructScan(dbRow)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return dbRow.ToSongRequest(), true, nil
}

func (r *SongRepository) ListUserSongRequests(userID, page, limit int) ([]models.SongRequest, int, error) {
	offset := (page - 1) * limit
	var total int
	if err := r.db.QueryRowx(`SELECT COUNT(*) FROM song_requests WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	dbRows := []models.SongRequestRow{}
	err := r.db.Select(&dbRows,
		`SELECT id, user_id, song_title, reference_link, lyrics_type, lyrics, status, admin_notes, "createdAt", "updatedAt"
		 FROM song_requests WHERE user_id = $1 ORDER BY "createdAt" DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]models.SongRequest, len(dbRows))
	for i, dbRow := range dbRows {
		rows[i] = *dbRow.ToSongRequest()
	}
	return rows, total, nil
}

func (r *SongRepository) DeleteSongRequest(id, userID int) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM song_requests WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ── Song Reports ──────────────────────────────────────────────────────────────

func (r *SongRepository) CreateSongReport(userID, songID int, reportType, description, evidenceURL string) (*models.SongReport, error) {
	report := &models.SongReportRow{}
	err := r.db.QueryRowx(
		`INSERT INTO song_reports (user_id, song_id, report_type, description, evidence_url)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, song_id,
			(SELECT title FROM songs WHERE id = $2) AS song_title,
			report_type, description, evidence_url, status, admin_notes, "createdAt", "updatedAt"`,
		userID, songID, reportType, description, evidenceURL,
	).StructScan(report)
	if err != nil {
		return nil, err
	}
	return report.ToSongReport(), nil
}

func (r *SongRepository) ListSongReports(status string, page, limit int) ([]models.SongReport, int, error) {
	offset := (page - 1) * limit
	where := "1=1"
	args := []any{}

	if status != "" {
		where = "sr.status = $1"
		args = append(args, status)
	}

	var total int
	countQ := `SELECT COUNT(*) FROM song_reports sr WHERE ` + where
	if err := r.db.QueryRowx(r.db.Rebind(countQ), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dbRows := []models.SongReportRow{}
	args = append(args, limit, offset)
	q := r.db.Rebind(`SELECT sr.id, sr.user_id, sr.song_id, s.title AS song_title, sr.report_type,
		sr.description, sr.evidence_url, sr.status, sr.admin_notes, sr."createdAt", sr."updatedAt"
		FROM song_reports sr
		JOIN songs s ON s.id = sr.song_id
		WHERE ` + where + ` ORDER BY sr."createdAt" DESC LIMIT ? OFFSET ?`)
	if err := r.db.Select(&dbRows, q, args...); err != nil {
		return nil, 0, err
	}

	rows := make([]models.SongReport, len(dbRows))
	for i, dbRow := range dbRows {
		rows[i] = *dbRow.ToSongReport()
	}
	return rows, total, nil
}

func (r *SongRepository) UpdateSongReportStatus(id int, status, adminNotes string) error {
	_, err := r.db.Exec(
		`UPDATE song_reports SET status = $1, admin_notes = $2, "updatedAt" = NOW() WHERE id = $3`,
		status, adminNotes, id,
	)
	return err
}

func (r *SongRepository) GetSongReportByID(id int) (*models.SongReport, bool, error) {
	dbRow := &models.SongReportRow{}
	err := r.db.QueryRowx(
		`SELECT sr.id, sr.user_id, sr.song_id, s.title AS song_title, sr.report_type,
		 sr.description, sr.evidence_url, sr.status, sr.admin_notes, sr."createdAt", sr."updatedAt"
		 FROM song_reports sr
		 JOIN songs s ON s.id = sr.song_id
		 WHERE sr.id = $1`,
		id,
	).StructScan(dbRow)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return dbRow.ToSongReport(), true, nil
}

func (r *SongRepository) ListUserSongReports(userID, page, limit int) ([]models.SongReport, int, error) {
	offset := (page - 1) * limit
	var total int
	if err := r.db.QueryRowx(`SELECT COUNT(*) FROM song_reports WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	dbRows := []models.SongReportRow{}
	err := r.db.Select(&dbRows,
		`SELECT sr.id, sr.user_id, sr.song_id, s.title AS song_title, sr.report_type,
		 sr.description, sr.evidence_url, sr.status, sr.admin_notes, sr."createdAt", sr."updatedAt"
		 FROM song_reports sr
		 JOIN songs s ON s.id = sr.song_id
		 WHERE sr.user_id = $1
		 ORDER BY sr."createdAt" DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]models.SongReport, len(dbRows))
	for i, dbRow := range dbRows {
		rows[i] = *dbRow.ToSongReport()
	}
	return rows, total, nil
}

func (r *SongRepository) DeleteSongReport(id, userID int) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM song_reports WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
