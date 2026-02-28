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

func (r *UserRepository) CountEligible(search string) (int, error) {
	where := `WHERE CAST(userlevel AS SIGNED) > 2`
	args := []any{}
	if search != "" {
		where += ` AND (email LIKE ? OR nama LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	var total int
	err := r.db.Get(&total, `SELECT COUNT(*) FROM peserta `+where, args...)
	return total, err
}

func (r *UserRepository) ListEligible(search string, page, limit int) ([]models.Peserta, error) {
	offset := (page - 1) * limit
	where := `WHERE CAST(userlevel AS SIGNED) > 2`
	args := []any{}
	if search != "" {
		where += ` AND (email LIKE ? OR nama LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	query := `SELECT id_peserta,nama,email,password,usercode,userlevel,verifikasi,status,role FROM peserta ` + where + ` ORDER BY id_peserta DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows := []models.Peserta{}
	err := r.db.Select(&rows, query, args...)
	return rows, err
}
