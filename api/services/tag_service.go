package services

import (
	"strings"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type TagService struct {
	repo *repositories.TagRepository
}

func NewTagService(repo *repositories.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) List(search string) ([]map[string]any, error) {
	rows, err := s.repo.List(search)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{"id": r.ID, "name": r.Name, "description": utils.NullableString(r.Description)})
	}
	return out, nil
}

func (s *TagService) GetOrCreate(name string) (map[string]any, bool, error) {
	name = strings.TrimSpace(name)
	t, err := s.repo.FindByName(name)
	if err != nil {
		return nil, false, err
	}
	if t != nil {
		return map[string]any{"id": t.ID, "name": t.Name, "description": utils.NullableString(t.Description)}, false, nil
	}
	id, err := s.repo.Create(name)
	if err != nil {
		return nil, false, err
	}
	return map[string]any{"id": id, "name": name}, true, nil
}
