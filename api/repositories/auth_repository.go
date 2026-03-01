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

func (r *AuthRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.db.Get(&u, r.db.Rebind(`SELECT id,name,email,password,avatar_url,role,provider,provider_id,verified,status,"createdAt","updatedAt" FROM users WHERE email=? LIMIT 1`), email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepository) FindByID(id int) (*models.UserBasic, error) {
	var u models.UserBasic
	err := r.db.Get(&u, r.db.Rebind(`SELECT id,name,email FROM users WHERE id=?`), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindOrCreateGoogleUser upserts a user authenticated via Google OAuth.
func (r *AuthRepository) FindOrCreateGoogleUser(email, name, providerID string) (*models.User, error) {
	existing, err := r.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	_, err = r.db.Exec(
		r.db.Rebind(`INSERT INTO users (name, email, role, provider, provider_id, verified, status)
		 VALUES (?, ?, 'user', 'google', ?, TRUE, 'active')`),
		name, email, providerID,
	)
	if err != nil {
		return nil, err
	}
	return r.FindByEmail(email)
}

// CreateLocal registers a new local (email+password) user. Password must already be hashed.
func (r *AuthRepository) CreateLocal(name, email, hashedPassword string) (*models.User, error) {
	userCode := uuid.NewString()[:8]
	_ = userCode // kept for potential future use
	_, err := r.db.Exec(
		r.db.Rebind(`INSERT INTO users (name, email, password, role, provider, verified, status)
		 VALUES (?, ?, ?, 'user', 'local', FALSE, 'active')`),
		name, email, hashedPassword,
	)
	if err != nil {
		return nil, err
	}
	return r.FindByEmail(email)
}
