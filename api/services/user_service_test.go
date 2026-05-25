package services

import (
	"database/sql"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── List ──────────────────────────────────────────────────────────────────────

func TestUserService_List_Success(t *testing.T) {
	mockRepo := &mocks.UserRepo{
		CountFn: func(string) (int, error) { return 25, nil },
		ListFn: func(search string, page, limit int) ([]models.User, error) {
			return []models.User{
				{ID: 1, Username: "Alice", Email: "a@example.com"},
				{ID: 2, Username: "Bob", Email: "b@example.com"},
				{ID: 3, Username: "Carol", Email: "c@example.com"},
			}, nil
		},
	}
	svc := &UserService{repo: mockRepo}
	users, total, err := svc.List("", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, users, 3)
}

func TestUserService_List_PageNormalization(t *testing.T) {
	var gotPage, gotLimit int
	mockRepo := &mocks.UserRepo{
		CountFn: func(string) (int, error) { return 0, nil },
		ListFn: func(_ string, page, limit int) ([]models.User, error) {
			gotPage = page
			gotLimit = limit
			return nil, nil
		},
	}
	svc := &UserService{repo: mockRepo}
	// page=0 → clamped to 1; limit=0 → clamped to 20
	svc.List("", 0, 0)
	assert.Equal(t, 1, gotPage)
	assert.Equal(t, 20, gotLimit)
}

func TestUserService_List_LimitCapped(t *testing.T) {
	var gotLimit int
	mockRepo := &mocks.UserRepo{
		CountFn: func(string) (int, error) { return 0, nil },
		ListFn: func(_ string, _, limit int) ([]models.User, error) {
			gotLimit = limit
			return nil, nil
		},
	}
	svc := &UserService{repo: mockRepo}
	svc.List("", 1, 999) // above 100 → clamped to 20
	assert.Equal(t, 20, gotLimit)
}

// ── UpdateProfile ─────────────────────────────────────────────────────────────

func TestUserService_UpdateProfile_Success(t *testing.T) {
	upsertCalled := false
	mockRepo := &mocks.UserRepo{
		UpsertDetailFn: func(int, *string, *string, *string, *string) error {
			upsertCalled = true
			return nil
		},
	}
	svc := &UserService{repo: mockRepo}
	name := "Alice Smith"
	err := svc.UpdateProfile(1, &name, nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, upsertCalled)
}

// ── UpdateAvatarURL ───────────────────────────────────────────────────────────

func TestUserService_UpdateAvatarURL_Success(t *testing.T) {
	var gotURL string
	mockRepo := &mocks.UserRepo{
		UpdateAvatarURLFn: func(_ int, url string) error {
			gotURL = url
			return nil
		},
	}
	svc := &UserService{repo: mockRepo}
	err := svc.UpdateAvatarURL(5, "https://example.com/avatar.jpg")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/avatar.jpg", gotURL)
}

// ── GetAvatarURL ──────────────────────────────────────────────────────────────

func TestUserService_GetAvatarURL_Set(t *testing.T) {
	mockRepo := &mocks.UserRepo{
		FindByIDFn: func(int) (*models.User, error) {
			return &models.User{AvatarURL: sql.NullString{String: "https://img.com/x.jpg", Valid: true}}, nil
		},
	}
	svc := &UserService{repo: mockRepo}
	url, err := svc.GetAvatarURL(1)
	require.NoError(t, err)
	assert.Equal(t, "https://img.com/x.jpg", url)
}

func TestUserService_GetAvatarURL_Null(t *testing.T) {
	mockRepo := &mocks.UserRepo{
		FindByIDFn: func(int) (*models.User, error) {
			return &models.User{AvatarURL: sql.NullString{Valid: false}}, nil
		},
	}
	svc := &UserService{repo: mockRepo}
	url, err := svc.GetAvatarURL(1)
	require.NoError(t, err)
	assert.Empty(t, url)
}

// ── RequestDeletion ───────────────────────────────────────────────────────────

func TestUserService_RequestDeletion_UserFound(t *testing.T) {
	var receivedUserID *int
	mockRepo := &mocks.UserRepo{
		FindByEmailFn: func(string) (*models.User, error) {
			return &models.User{ID: 5}, nil
		},
		CreateDeletionRequestFn: func(_ string, userID *int, _ string) error {
			receivedUserID = userID
			return nil
		},
	}
	svc := &UserService{repo: mockRepo}
	err := svc.RequestDeletion("alice@example.com", "1.2.3.4")
	require.NoError(t, err)
	require.NotNil(t, receivedUserID)
	assert.Equal(t, 5, *receivedUserID)
}

func TestUserService_RequestDeletion_UserNotFound(t *testing.T) {
	var receivedUserID *int
	mockRepo := &mocks.UserRepo{
		FindByEmailFn: func(string) (*models.User, error) { return nil, nil },
		CreateDeletionRequestFn: func(_ string, userID *int, _ string) error {
			receivedUserID = userID
			return nil
		},
	}
	svc := &UserService{repo: mockRepo}
	err := svc.RequestDeletion("anon@example.com", "1.2.3.4")
	require.NoError(t, err)
	assert.Nil(t, receivedUserID, "userID must be nil for anonymous deletion requests")
}
