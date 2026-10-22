package services

import (
	"database/sql"
	"fmt"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/repositories"
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
