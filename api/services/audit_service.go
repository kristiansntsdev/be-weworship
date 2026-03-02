package services

import "be-songbanks-v1/api/repositories"

type AuditService struct {
	repo *repositories.AuditRepository
}

func NewAuditService(repo *repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(userID *int, userName, userEmail, action, entityType string, entityID *int, entityName string, changes any) {
	s.repo.Log(userID, userName, userEmail, action, entityType, entityID, entityName, changes)
}

func (s *AuditService) List(action, entityType string, userID *int, page, limit int) ([]repositories.AuditLogRow, int, error) {
	return s.repo.List(action, entityType, userID, page, limit)
}
