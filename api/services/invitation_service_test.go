package services

import (
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationService_SearchUsers_NormalizesWeWorshipHandle(t *testing.T) {
	var gotSearch string
	mockUsers := &mocks.UserRepo{
		ListFn: func(search string, page, limit int) ([]models.User, error) {
			gotSearch = search
			assert.Equal(t, 1, page)
			assert.Equal(t, 10, limit)
			return []models.User{{ID: 5, Username: "KristianSantoso", Email: "ktian27@gmail.com"}}, nil
		},
	}
	svc := &InvitationService{users: mockUsers}

	users, err := svc.SearchUsers("ww@KristianSantoso", 1)

	require.NoError(t, err)
	assert.Equal(t, "KristianSantoso", gotSearch)
	require.Len(t, users, 1)
	assert.Equal(t, models.UserBasic{ID: 5, Username: "KristianSantoso", Email: "ktian27@gmail.com"}, users[0])
}

func TestInvitationService_SearchUsers_ExcludesRequestingUser(t *testing.T) {
	mockUsers := &mocks.UserRepo{
		ListFn: func(string, int, int) ([]models.User, error) {
			return []models.User{
				{ID: 5, Username: "KristianSantoso", Email: "ktian27@gmail.com"},
				{ID: 7, Username: "KristianE", Email: "k@example.com"},
			}, nil
		},
	}
	svc := &InvitationService{users: mockUsers}

	users, err := svc.SearchUsers("Kristian", 5)

	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, 7, users[0].ID)
}

func TestInvitationService_SearchUsers_HandlePrefixOnlyIsEmpty(t *testing.T) {
	mockUsers := &mocks.UserRepo{
		ListFn: func(string, int, int) ([]models.User, error) {
			t.Fatal("List must not be called for an empty handle search")
			return nil, nil
		},
	}
	svc := &InvitationService{users: mockUsers}

	users, err := svc.SearchUsers("ww@", 1)

	require.NoError(t, err)
	assert.Empty(t, users)
}
