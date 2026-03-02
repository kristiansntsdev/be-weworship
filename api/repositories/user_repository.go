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

func (r *UserRepository) GetDetail(userID int) (*models.UserDetail, error) {
	var d models.UserDetail
	err := r.db.Get(&d, `SELECT user_id,full_name,province,city,postal_code,"createdAt","updatedAt" FROM users_detail WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *UserRepository) UpsertDetail(userID int, fullName, province, city, postalCode *string) error {
	_, err := r.db.Exec(
		`INSERT INTO users_detail (user_id, full_name, province, city, postal_code, "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   full_name   = EXCLUDED.full_name,
		   province    = EXCLUDED.province,
		   city        = EXCLUDED.city,
		   postal_code = EXCLUDED.postal_code,
		   "updatedAt" = NOW()`,
		userID, fullName, province, city, postalCode,
	)
	return err
}
