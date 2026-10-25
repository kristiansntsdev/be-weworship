package repositories

import (
	"database/sql"

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
		where += ` AND (email ILIKE ? ESCAPE '\' OR username ILIKE ? ESCAPE '\')`
		like := containsPattern(search)
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
		where += ` AND (email ILIKE ? ESCAPE '\' OR username ILIKE ? ESCAPE '\')`
		like := containsPattern(search)
		args = append(args, like, like)
	}
	query := r.db.Rebind(`SELECT id,username,email,avatar_url,role,provider,verified,status,"createdAt","updatedAt" FROM users ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`)
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

func (r *UserRepository) FindByID(userID int) (*models.User, error) {
	var u models.User
	err := r.db.Get(&u, `SELECT id, username, email, role, provider, verified, status, avatar_url, "createdAt", "updatedAt" FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateRole(userID int, role string) (*models.User, error) {
	var u models.User
	err := r.db.Get(
		&u,
		`UPDATE users SET role = $1, "updatedAt" = NOW() WHERE id = $2 RETURNING id, username, email, role, provider, verified, status, avatar_url, "createdAt", "updatedAt"`,
		role,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateVerified(userID int, verified bool) (*models.User, error) {
	var u models.User
	err := r.db.Get(
		&u,
		`UPDATE users SET verified = $1, "updatedAt" = NOW() WHERE id = $2 RETURNING id, username, email, role, provider, verified, status, avatar_url, "createdAt", "updatedAt"`,
		verified,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) CreateVerificationRequest(userID int, displayName, proofLink string, note *string) (*models.VerificationRequest, error) {
	var row models.VerificationRequest
	err := r.db.Get(
		&row,
		`INSERT INTO verification_requests (user_id, display_name, proof_link, note)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, display_name, proof_link, note, status, admin_notes, "createdAt", "updatedAt"`,
		userID, displayName, proofLink, note,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *UserRepository) GetLatestVerificationRequest(userID int) (*models.VerificationRequest, error) {
	var row models.VerificationRequest
	err := r.db.Get(
		&row,
		`SELECT id, user_id, display_name, proof_link, note, status, admin_notes, "createdAt", "updatedAt"
		 FROM verification_requests WHERE user_id = $1 ORDER BY "createdAt" DESC LIMIT 1`,
		userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *UserRepository) HasPendingVerificationRequest(userID int) (bool, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM verification_requests WHERE user_id = $1 AND status = 'pending'`, userID)
	return n > 0, err
}

func (r *UserRepository) CountVerificationRequests(status string) (int, error) {
	where := `WHERE 1=1`
	args := []any{}
	if status != "" {
		where += ` AND status = ?`
		args = append(args, status)
	}
	var total int
	err := r.db.Get(&total, r.db.Rebind(`SELECT COUNT(*) FROM verification_requests `+where), args...)
	return total, err
}

func (r *UserRepository) ListVerificationRequests(status string, page, limit int) ([]models.VerificationRequest, error) {
	offset := (page - 1) * limit
	where := `WHERE 1=1`
	args := []any{}
	if status != "" {
		where += ` AND vr.status = ?`
		args = append(args, status)
	}
	query := r.db.Rebind(`
		SELECT vr.id, vr.user_id, vr.display_name, vr.proof_link, vr.note, vr.status, vr.admin_notes,
		       vr."createdAt", vr."updatedAt", u.username, u.email
		FROM verification_requests vr
		JOIN users u ON u.id = vr.user_id
		` + where + ` ORDER BY vr."createdAt" DESC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)
	rows := []models.VerificationRequest{}
	err := r.db.Select(&rows, query, args...)
	return rows, err
}

func (r *UserRepository) GetVerificationRequestByID(id int) (*models.VerificationRequest, error) {
	var row models.VerificationRequest
	err := r.db.Get(
		&row,
		`SELECT id, user_id, display_name, proof_link, note, status, admin_notes, "createdAt", "updatedAt"
		 FROM verification_requests WHERE id = $1`,
		id,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *UserRepository) UpdateVerificationRequestStatus(id int, status, adminNotes string) error {
	_, err := r.db.Exec(
		`UPDATE verification_requests SET status = $1, admin_notes = $2, "updatedAt" = NOW() WHERE id = $3`,
		status, adminNotes, id,
	)
	return err
}

func (r *UserRepository) UpdateAvatarURL(userID int, avatarURL string) error {
	_, err := r.db.Exec(`UPDATE users SET avatar_url = $1, "updatedAt" = NOW() WHERE id = $2`, avatarURL, userID)
	return err
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.db.Get(&u, `SELECT id,username,email,role,provider,verified,status,"createdAt","updatedAt" FROM users WHERE email = $1 LIMIT 1`, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepository) CreateDeletionRequest(email string, userID *int, ipAddress string) error {
	_, err := r.db.Exec(
		`INSERT INTO deletion_requests (email, user_id, ip_address) VALUES ($1, $2, $3)`,
		email, userID, ipAddress,
	)
	return err
}
