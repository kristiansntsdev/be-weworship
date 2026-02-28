package repositories

import (
	"database/sql"

	"be-songbanks-v1/api/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AuthRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindPengurusByUsername(username string) (*models.Pengurus, error) {
	var p models.Pengurus
	err := r.db.Get(&p, `SELECT id_pengurus,nama,username,password,leveladmin,nowa,kotalevelup FROM pengurus WHERE username=? LIMIT 1`, username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AuthRepository) FindPesertaByEmail(email string) (*models.Peserta, error) {
	var p models.Peserta
	err := r.db.Get(&p, `SELECT id_peserta,nama,email,password,usercode,userlevel,verifikasi,status,role FROM peserta WHERE email=? LIMIT 1`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AuthRepository) FindPesertaBasicByID(id int) (*models.PesertaBasic, error) {
	var p models.PesertaBasic
	err := r.db.Get(&p, `SELECT id_peserta,nama,email FROM peserta WHERE id_peserta=?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindOrCreateGoogleUser looks up a peserta by email and creates one if not found.
// Google-created accounts are auto-verified (verifikasi=1) with userlevel=3.
func (r *AuthRepository) FindOrCreateGoogleUser(email, name string) (*models.Peserta, error) {
	existing, err := r.FindPesertaByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	userCode := uuid.NewString()[:8]
	randomPassword := uuid.NewString() // Google users never log in with a password

	_, err = r.db.Exec(
		`INSERT INTO peserta (nama, email, password, usercode, userlevel, verifikasi, status, role)
		 VALUES (?, ?, ?, ?, '3', '1', 'active', 'member')`,
		name, email, randomPassword, userCode,
	)
	if err != nil {
		return nil, err
	}

	return r.FindPesertaByEmail(email)
}
