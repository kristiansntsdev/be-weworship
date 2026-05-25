package services

import (
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── List ──────────────────────────────────────────────────────────────────────

func TestTagService_List_Success(t *testing.T) {
	mockRepo := &mocks.TagRepo{
		ListFn: func(search string) ([]models.Tag, error) {
			return []models.Tag{
				{ID: 1, Name: "worship"},
				{ID: 2, Name: "praise"},
			}, nil
		},
	}
	svc := NewTagService(nil) // constructor accepts *TagRepository
	svc.repo = mockRepo       // override with interface

	rows, err := svc.List("wor")
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, "worship", rows[0]["name"])
}

func TestTagService_List_Empty(t *testing.T) {
	mockRepo := &mocks.TagRepo{
		ListFn: func(string) ([]models.Tag, error) { return nil, nil },
	}
	svc := &TagService{repo: mockRepo}
	rows, err := svc.List("")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// ── GetOrCreate ───────────────────────────────────────────────────────────────

func TestTagService_GetOrCreate_Existing(t *testing.T) {
	createCalled := false
	mockRepo := &mocks.TagRepo{
		FindByNameFn: func(name string) (*models.Tag, error) {
			return &models.Tag{ID: 3, Name: "worship"}, nil
		},
		CreateFn: func(string) (int, error) {
			createCalled = true
			return 0, nil
		},
	}
	svc := &TagService{repo: mockRepo}
	result, created, err := svc.GetOrCreate("worship")

	require.NoError(t, err)
	assert.False(t, created)
	assert.False(t, createCalled, "Create must NOT be called when tag already exists")
	assert.Equal(t, 3, result["id"])
}

func TestTagService_GetOrCreate_New(t *testing.T) {
	mockRepo := &mocks.TagRepo{
		FindByNameFn: func(string) (*models.Tag, error) { return nil, nil },
		CreateFn:     func(string) (int, error) { return 7, nil },
	}
	svc := &TagService{repo: mockRepo}
	result, created, err := svc.GetOrCreate("newTag")

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, 7, result["id"])
	assert.Equal(t, "newTag", result["name"]) // TrimSpace preserves case — only whitespace is stripped
}

func TestTagService_GetOrCreate_WhitespaceTrimmed(t *testing.T) {
	receivedName := ""
	mockRepo := &mocks.TagRepo{
		FindByNameFn: func(name string) (*models.Tag, error) {
			receivedName = name
			return nil, nil
		},
		CreateFn: func(name string) (int, error) { return 1, nil },
	}
	svc := &TagService{repo: mockRepo}
	_, _, err := svc.GetOrCreate("  worship  ")
	require.NoError(t, err)
	assert.Equal(t, "worship", receivedName, "leading/trailing whitespace must be trimmed")
}

func TestTagService_GetOrCreate_EmptyName(t *testing.T) {
	// TrimSpace("") == "" — FindByName is still called with "". Behaviour
	// depends on repo; here we test the service doesn't panic and returns.
	mockRepo := &mocks.TagRepo{
		FindByNameFn: func(string) (*models.Tag, error) { return nil, nil },
		CreateFn:     func(string) (int, error) { return 1, nil },
	}
	svc := &TagService{repo: mockRepo}
	_, _, err := svc.GetOrCreate("")
	require.NoError(t, err) // no service-level validation on empty name
}
