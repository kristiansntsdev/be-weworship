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
		query += ` WHERE name LIKE ? OR description LIKE ?`
		like := "%" + search + "%"
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
