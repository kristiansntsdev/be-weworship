package services

import (
	"database/sql"
	"fmt"
	"strings"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type UserService struct {
	repo repositories.UserRepoIface
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(search string, page, limit int) ([]models.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	total, err := s.repo.Count(search)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.List(search, page, limit)
	return rows, total, err
}

func (s *UserService) GetDetail(userID int) (*models.UserDetail, error) {
	return s.repo.GetDetail(userID)
}

func (s *UserService) UpdateProfile(userID int, fullName, province, city, postalCode *string) error {
	existing, err := s.repo.GetDetail(userID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	fn := mergeDetailField(fullName, existing, func(d *models.UserDetail) sql.NullString { return d.FullName })
	prov := mergeDetailField(province, existing, func(d *models.UserDetail) sql.NullString { return d.Province })
	cityVal := mergeDetailField(city, existing, func(d *models.UserDetail) sql.NullString { return d.City })
	postal := mergeDetailField(postalCode, existing, func(d *models.UserDetail) sql.NullString { return d.PostalCode })

	return s.repo.UpsertDetail(userID, fn, prov, cityVal, postal)
}

func mergeDetailField(in *string, existing *models.UserDetail, getter func(*models.UserDetail) sql.NullString) *string {
	if in != nil {
		return in
	}
	if existing == nil {
		return nil
	}
	current := getter(existing)
	if !current.Valid {
		return nil
	}
	value := current.String
	return &value
}

func (s *UserService) UpdateAvatarURL(userID int, avatarURL string) error {
	return s.repo.UpdateAvatarURL(userID, avatarURL)
}

func (s *UserService) UpdateRole(userID int, role string) (*models.User, int, error) {
	if userID < 1 {
		return nil, 400, fmt.Errorf("invalid user id")
	}
	if role != "user" && role != "maintainer" {
		return nil, 400, fmt.Errorf("role must be 'user' or 'maintainer'")
	}
	u, err := s.repo.UpdateRole(userID, role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 404, fmt.Errorf("user not found")
		}
		return nil, 500, err
	}
	return u, 200, nil
}

func (s *UserService) GetAvatarURL(userID int) (string, error) {
	u, err := s.repo.FindByID(userID)
	if err != nil {
		return "", err
	}
	if u.AvatarURL.Valid {
		return u.AvatarURL.String, nil
	}
	return "", nil
}

func (s *UserService) RequestDeletion(email, ipAddress string) error {
	var userID *int
	u, err := s.repo.FindByEmail(email)
	if err != nil {
		return err
	}
	if u != nil {
		userID = &u.ID
	}
	return s.repo.CreateDeletionRequest(email, userID, ipAddress)
}

func (s *UserService) UpdateVerified(userID int, verified bool) (*models.User, int, error) {
	if userID < 1 {
		return nil, 400, fmt.Errorf("invalid user id")
	}
	u, err := s.repo.UpdateVerified(userID, verified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 404, fmt.Errorf("user not found")
		}
		return nil, 500, err
	}
	return u, 200, nil
}

func (s *UserService) CreateVerificationRequest(userID int, displayName, proofLink, note string) (map[string]any, int, error) {
	displayName = strings.TrimSpace(displayName)
	proofLink = strings.TrimSpace(proofLink)
	note = strings.TrimSpace(note)
	if displayName == "" {
		return nil, 400, fmt.Errorf("display_name is required")
	}
	if len(displayName) > 120 {
		return nil, 400, fmt.Errorf("display_name is too long")
	}
	if !isHTTPURL(proofLink) {
		return nil, 400, fmt.Errorf("proof_link must be an http(s) URL")
	}

	u, err := s.repo.FindByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 404, fmt.Errorf("user not found")
		}
		return nil, 500, err
	}
	if u.Verified {
		return nil, 400, fmt.Errorf("account is already verified")
	}
	pending, err := s.repo.HasPendingVerificationRequest(userID)
	if err != nil {
		return nil, 500, err
	}
	if pending {
		return nil, 409, fmt.Errorf("a verification request is already pending")
	}

	var notePtr *string
	if note != "" {
		notePtr = &note
	}
	row, err := s.repo.CreateVerificationRequest(userID, displayName, proofLink, notePtr)
	if err != nil {
		return nil, 500, err
	}
	return mapVerificationRequest(row), 201, nil
}

func (s *UserService) GetMyVerificationRequest(userID int) (map[string]any, error) {
	row, err := s.repo.GetLatestVerificationRequest(userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return mapVerificationRequest(row), nil
}

func (s *UserService) ListVerificationRequests(status string, page, limit int) ([]map[string]any, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && status != "pending" && status != "approved" && status != "rejected" {
		return nil, 0, fmt.Errorf("invalid status")
	}
	total, err := s.repo.CountVerificationRequests(status)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.ListVerificationRequests(status, page, limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, len(rows))
	for i := range rows {
		out[i] = mapVerificationRequest(&rows[i])
	}
	return out, total, nil
}

func (s *UserService) UpdateVerificationRequestStatus(id int, status, adminNotes string) (int, error) {
	if id < 1 {
		return 400, fmt.Errorf("invalid request id")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "pending" && status != "approved" && status != "rejected" {
		return 400, fmt.Errorf("status must be pending, approved, or rejected")
	}
	row, err := s.repo.GetVerificationRequestByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 404, fmt.Errorf("verification request not found")
		}
		return 500, err
	}
	if err := s.repo.UpdateVerificationRequestStatus(id, status, strings.TrimSpace(adminNotes)); err != nil {
		return 500, err
	}
	if status == "approved" {
		if _, err := s.repo.UpdateVerified(row.UserID, true); err != nil {
			return 500, err
		}
	}
	return 200, nil
}

func mapVerificationRequest(r *models.VerificationRequest) map[string]any {
	m := map[string]any{
		"id":           r.ID,
		"user_id":      r.UserID,
		"display_name": r.DisplayName,
		"proof_link":   r.ProofLink,
		"note":         utils.NullableString(r.Note),
		"status":       r.Status,
		"admin_notes":  utils.NullableString(r.AdminNotes),
		"createdAt":    r.CreatedAt,
		"updatedAt":    r.UpdatedAt,
	}
	if r.Username != "" {
		m["username"] = r.Username
	}
	if r.Email != "" {
		m["email"] = r.Email
	}
	return m
}
