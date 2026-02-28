package services

import (
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(search string, page, limit int) ([]map[string]any, map[string]any, error) {
	total, err := s.repo.CountEligible(search)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.repo.ListEligible(search, page, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id_peserta": r.ID,
			"usercode":   r.UserCode,
			"nama":       r.Nama,
			"email":      r.Email,
			"userlevel":  r.UserLevel,
			"status":     r.Status,
			"role":       r.Role,
		})
	}
	pagination := map[string]any{"currentPage": page, "totalPages": utils.Ceil(total, limit), "totalItems": total, "itemsPerPage": limit}
	return out, pagination, nil
}
