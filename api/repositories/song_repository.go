package repositories

import (
	"database/sql"
	"strings"

	"be-songbanks-v1/api/models"
	"github.com/jmoiron/sqlx"
)

type SongRepository struct {
	db *sqlx.DB
}

func NewSongRepository(db *sqlx.DB) *SongRepository {
	return &SongRepository{db: db}
}

func (r *SongRepository) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int) ([]models.Song, int, error) {
	offset := (page - 1) * limit
	where := []string{"1=1"}
	args := []any{}

	if strings.TrimSpace(search) != "" {
		where = append(where, `(s.title LIKE ? OR s.artist LIKE ? OR s.lyrics_and_chords LIKE ?)`)
		like := "%" + strings.TrimSpace(search) + "%"
		args = append(args, like, like, like)
	}
	if strings.TrimSpace(baseChord) != "" {
		where = append(where, `s.base_chord LIKE ?`)
		args = append(args, "%"+strings.TrimSpace(baseChord)+"%")
	}
	if len(tagIDs) > 0 {
		ph := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
		where = append(where, `s.id IN (SELECT DISTINCT song_id FROM song_tags WHERE tag_id IN (`+ph+`))`)
		for _, id := range tagIDs {
			args = append(args, id)
		}
	}

	whereClause := strings.Join(where, " AND ")
	countArgs := append([]any{}, args...)
	var total int
	if err := r.db.Get(&total, `SELECT COUNT(DISTINCT s.id) FROM songs s WHERE `+whereClause, countArgs...); err != nil {
		return nil, 0, err
	}

	query := `SELECT s.id,s.title,s.artist,s.base_chord,s.lyrics_and_chords,s.createdAt,s.updatedAt FROM songs s WHERE ` + whereClause + ` ORDER BY ` + sortBy + ` ` + sortOrder + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows := []models.Song{}
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *SongRepository) GetByID(id int) (*models.Song, error) {
	var row models.Song
	err := r.db.Get(&row, `SELECT id,title,artist,base_chord,lyrics_and_chords,createdAt,updatedAt FROM songs WHERE id=?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *SongRepository) Create(title, artistJSON string, baseChord, lyrics *string) (int, error) {
	res, err := r.db.Exec(`INSERT INTO songs (title,artist,base_chord,lyrics_and_chords,createdAt,updatedAt) VALUES (?,?,?,?,NOW(),NOW())`, title, artistJSON, baseChord, lyrics)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (r *SongRepository) UpdateFields(id int, setExpr string, args ...any) (int64, error) {
	query := `UPDATE songs SET ` + setExpr + `,updatedAt=NOW() WHERE id=?`
	args = append(args, id)
	res, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *SongRepository) DeleteByID(id int) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM songs WHERE id=?`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *SongRepository) ClearSongTags(songID int) error {
	_, err := r.db.Exec(`DELETE FROM song_tags WHERE song_id=?`, songID)
	return err
}

func (r *SongRepository) AssignSongTag(songID, tagID int) error {
	_, err := r.db.Exec(`INSERT IGNORE INTO song_tags (song_id,tag_id,createdAt,updatedAt) VALUES (?,?,NOW(),NOW())`, songID, tagID)
	return err
}

func (r *SongRepository) ExistsByID(id int) (bool, error) {
	var count int
	if err := r.db.Get(&count, `SELECT COUNT(*) FROM songs WHERE id=?`, id); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SongRepository) ListArtistsRaw() ([]string, error) {
	rows := []string{}
	err := r.db.Select(&rows, `SELECT artist FROM songs WHERE artist IS NOT NULL AND artist != ''`)
	return rows, err
}
