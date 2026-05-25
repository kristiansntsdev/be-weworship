package services

import (
	"testing"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── List ──────────────────────────────────────────────────────────────────────

func TestAuditService_List_Success(t *testing.T) {
	mockRepo := &mocks.AuditRepo{
		ListFn: func(action, entityType string, userID *int, page, limit int) ([]repositories.AuditLogRow, int, error) {
			return []repositories.AuditLogRow{
				{ID: 1, Action: "create", EntityType: "song"},
				{ID: 2, Action: "update", EntityType: "song"},
			}, 2, nil
		},
	}
	svc := &AuditService{repo: mockRepo}
	rows, total, err := svc.List("", "", nil, 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestAuditService_List_PassesThroughArguments(t *testing.T) {
	var gotAction, gotEntity string
	var gotPage, gotLimit int
	mockRepo := &mocks.AuditRepo{
		ListFn: func(action, entityType string, _ *int, page, limit int) ([]repositories.AuditLogRow, int, error) {
			gotAction = action
			gotEntity = entityType
			gotPage = page
			gotLimit = limit
			return nil, 0, nil
		},
	}
	svc := &AuditService{repo: mockRepo}
	svc.List("create", "song", nil, 2, 20)

	assert.Equal(t, "create", gotAction)
	assert.Equal(t, "song", gotEntity)
	assert.Equal(t, 2, gotPage)
	assert.Equal(t, 20, gotLimit)
}

// ── Log ───────────────────────────────────────────────────────────────────────

func TestAuditService_Log_CallsRepo(t *testing.T) {
	logCalled := false
	mockRepo := &mocks.AuditRepo{
		LogFn: func(*int, string, string, string, string, *int, string, any) {
			logCalled = true
		},
	}
	svc := &AuditService{repo: mockRepo}
	uid := 1
	entityID := 42
	svc.Log(&uid, "Admin", "admin@example.com", "create", "song", &entityID, "Amazing Grace", nil)
	assert.True(t, logCalled, "repo.Log must be called")
}
