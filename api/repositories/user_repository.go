package repositories

import (
	"be-songbanks-v1/api/models"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Count(search string) (int, error) {
	where := `WHERE 1=1`
	args := []any{}
	if search != "" {
		where += ` AND (email LIKE ? OR name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	var total int
	err := r.db.Get(&total, r.db.Rebind(`SELECT COUNT(*) FROM users `+where), args...)
	return total, err
}

func (r *UserRepository) List(search string, page, limit int) ([]models.User, error) {
	offset := (page - 1) * limit
	where := `WHERE 1=1`
	args := []any{}
	if search != "" {
		where += ` AND (email LIKE ? OR name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	query := r.db.Rebind(`SELECT id,name,email,avatar_url,role,provider,verified,status,"createdAt","updatedAt" FROM users ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)
	rows := []models.User{}
	err := r.db.Select(&rows, query, args...)
	return rows, err
}
