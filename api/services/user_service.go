package services

import (
"be-songbanks-v1/api/models"
"be-songbanks-v1/api/repositories"
)

type UserService struct {
repo *repositories.UserRepository
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
return s.repo.UpsertDetail(userID, fullName, province, city, postalCode)
}
