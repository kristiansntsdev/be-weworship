package services

import (
	"context"
	"testing"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── SaveDeviceToken ───────────────────────────────────────────────────────────

func TestNotificationService_SaveDeviceToken_Success(t *testing.T) {
	upsertCalled := false
	mockRepo := &mocks.NotificationRepo{
		UpsertDeviceTokenFn: func(int, string, string) error {
			upsertCalled = true
			return nil
		},
	}
	svc := &NotificationService{repo: mockRepo, push: &mocks.PushProvider{}}
	err := svc.SaveDeviceToken(1, "ExponentPushToken[abc]", "ios")
	require.NoError(t, err)
	assert.True(t, upsertCalled)
}

// ── RemoveDeviceToken ─────────────────────────────────────────────────────────

func TestNotificationService_RemoveDeviceToken_Success(t *testing.T) {
	deleteCalled := false
	mockRepo := &mocks.NotificationRepo{
		DeleteDeviceTokenFn: func(int, string) error {
			deleteCalled = true
			return nil
		},
	}
	svc := &NotificationService{repo: mockRepo, push: &mocks.PushProvider{}}
	err := svc.RemoveDeviceToken(1, "ExponentPushToken[abc]")
	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

// ── GetNotifications ──────────────────────────────────────────────────────────

func TestNotificationService_GetNotifications_Success(t *testing.T) {
	mockRepo := &mocks.NotificationRepo{
		ListByUserIDFn: func(int, int, int) ([]repositories.NotificationRow, error) {
			return []repositories.NotificationRow{
				{ID: 1, Title: "New Song"},
				{ID: 2, Title: "Playlist Updated"},
			}, nil
		},
	}
	svc := &NotificationService{repo: mockRepo, push: &mocks.PushProvider{}}
	rows, err := svc.GetNotifications(1, 1, 10)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

// ── MarkAsRead ────────────────────────────────────────────────────────────────

func TestNotificationService_MarkAsRead_Success(t *testing.T) {
	var gotID, gotUserID int
	mockRepo := &mocks.NotificationRepo{
		MarkReadFn: func(id, userID int) error {
			gotID = id
			gotUserID = userID
			return nil
		},
	}
	svc := &NotificationService{repo: mockRepo, push: &mocks.PushProvider{}}
	err := svc.MarkAsRead(5, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, gotID)
	assert.Equal(t, 3, gotUserID)
}

// ── GetUnreadCount ────────────────────────────────────────────────────────────

func TestNotificationService_GetUnreadCount_Success(t *testing.T) {
	mockRepo := &mocks.NotificationRepo{
		CountUnreadFn: func(int) (int, error) { return 3, nil },
	}
	svc := &NotificationService{repo: mockRepo, push: &mocks.PushProvider{}}
	count, err := svc.GetUnreadCount(1)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// ── NotifyNewSong ─────────────────────────────────────────────────────────────

func TestNotificationService_NotifyNewSong_CallsPushAndSavesRow(t *testing.T) {
	broadcastSaved := false
	pushSent := false

	mockRepo := &mocks.NotificationRepo{
		SaveBroadcastNotificationFn: func(string, string, string, string) error {
			broadcastSaved = true
			return nil
		},
		GetAllTokensFn: func() ([]string, error) {
			return []string{"ExponentPushToken[abc]"}, nil
		},
	}
	mockPush := &mocks.PushProvider{
		SendFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
			pushSent = true
			return nil
		},
	}
	svc := &NotificationService{repo: mockRepo, push: mockPush}
	svc.NotifyNewSong("Amazing Grace")

	assert.True(t, broadcastSaved)
	assert.True(t, pushSent)
}

func TestNotificationService_NotifyMemberLeft_ZeroOwnerID_Skips(t *testing.T) {
	saveCalled := false
	mockRepo := &mocks.NotificationRepo{
		SaveNotificationFn: func(int, string, string, string, string) error {
			saveCalled = true
			return nil
		},
	}
	svc := &NotificationService{repo: mockRepo, push: &mocks.PushProvider{}}
	svc.NotifyMemberLeft("Sunday Set", "Alice", 0) // ownerID=0 → skip
	assert.False(t, saveCalled, "must not save when ownerID is 0")
}
