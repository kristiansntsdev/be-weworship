package repositories

import (
	"database/sql"
	"fmt"
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

func (r *SongRepository) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, hasLink, chordPro *bool) ([]models.Song, int, error) {
	offset := (page - 1) * limit
	where := []string{"1=1"}
	args := []any{}

	if strings.TrimSpace(search) != "" {
		where = append(where, `(s.title ILIKE ? OR s.artist ILIKE ? OR s.lyrics_and_chords ILIKE ?)`)
		like := "%" + strings.TrimSpace(search) + "%"
		args = append(args, like, like, like)
	}
	if strings.TrimSpace(baseChord) != "" {
		where = append(where, `s.base_chord ILIKE ?`)
		args = append(args, "%"+strings.TrimSpace(baseChord)+"%")
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

	whereClause := strings.Join(where, " AND ")
	countArgs := append([]any{}, args...)
	var total int
	countQ := r.db.Rebind(`SELECT COUNT(DISTINCT s.id) FROM songs s WHERE ` + whereClause)
	if err := r.db.Get(&total, countQ, countArgs...); err != nil {
		return nil, 0, err
	}

	query := r.db.Rebind(`SELECT s.id,s.slug,s.title,s.artist,s.base_chord,s.bpm,s.lyrics_and_chords,s.external_links,s.dmca_takedown,s.dmca_status_notes,s."createdAt",s."updatedAt" FROM songs s WHERE ` + whereClause + ` ORDER BY ` + sortBy + ` ` + sortOrder + ` LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)
	rows := []models.Song{}
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
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
	var id int
	err := r.db.QueryRow(r.db.Rebind(`INSERT INTO songs (slug,title,artist,base_chord,bpm,lyrics_and_chords,external_links,dmca_takedown,dmca_status_notes,"createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?,?,NOW(),NOW()) RETURNING id`), baseSlug, title, artistJSON, baseChord, bpm, lyrics, externalLinks, dmcaTakedown, dmcaStatusNotes).Scan(&id)
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
	err := r.db.Select(&rows, `SELECT artist FROM songs WHERE artist IS NOT NULL AND artist != ''`)
	return rows, err
}

func (r *SongRepository) Count() (int, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM songs`)
	return n, err
}
