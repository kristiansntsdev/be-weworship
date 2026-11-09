package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"be-songbanks-v1/api/models"
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

// ── NotifySongRequest ─────────────────────────────────────────────────────────

func TestNotificationService_NotifySongRequest_SendsEscapedAlert(t *testing.T) {
	var sent string
	tg := &mocks.Telegram{
		SendMessageFn: func(ctx context.Context, text string) error {
			_, hasDeadline := ctx.Deadline()
			assert.True(t, hasDeadline, "Telegram call must carry a timeout")
			sent = text
			return nil
		},
	}
	svc := &NotificationService{telegram: tg, clientURL: "https://admin.example.com"}
	svc.NotifySongRequest(&models.SongRequest{
		ID:            7,
		UserID:        5,
		SongTitle:     "Holy <Spirit> & Fire",
		ReferenceLink: "https://example.com/song?a=1&b=2",
		LyricsType:    "lyrics_chords",
		Lyrics:        strings.Repeat("ab", 200),
	}, "kris")

	assert.Contains(t, sent, "<b>Title:</b> Holy &lt;Spirit&gt; &amp; Fire")
	assert.Contains(t, sent, "<b>From:</b> kris (user #5)")
	assert.Contains(t, sent, "<b>Type:</b> Lyrics + chords")
	assert.Contains(t, sent, "https://example.com/song?a=1&amp;b=2")
	assert.Contains(t, sent, strings.Repeat("ab", 150)+"…</pre>")
	assert.NotContains(t, sent, strings.Repeat("ab", 151))
	assert.Contains(t, sent, "https://admin.example.com/dashboard/song-requests")
}

func TestNotificationService_NotifySongRequest_NoLinkNoLyricsLines(t *testing.T) {
	var sent string
	tg := &mocks.Telegram{SendMessageFn: func(_ context.Context, text string) error { sent = text; return nil }}
	svc := &NotificationService{telegram: tg}
	svc.NotifySongRequest(&models.SongRequest{SongTitle: "Song", LyricsType: "lyrics"}, "")

	assert.Contains(t, sent, "<b>From:</b> unknown")
	assert.Contains(t, sent, "<b>Type:</b> Lyrics\n")
	assert.NotContains(t, sent, "<b>Link:</b>")
	assert.NotContains(t, sent, "<pre>")
	assert.NotContains(t, sent, "/dashboard/song-requests")
}

func TestNotificationService_NotifySongRequest_TelegramDisabled_Skips(t *testing.T) {
	svc := &NotificationService{} // telegram nil
	assert.NotPanics(t, func() {
		svc.NotifySongRequest(&models.SongRequest{SongTitle: "Song"}, "kris")
	})
}

func TestNotificationService_NotifySongRequest_SendError_DoesNotPanic(t *testing.T) {
	called := false
	tg := &mocks.Telegram{SendMessageFn: func(context.Context, string) error {
		called = true
		return errors.New("telegram sendMessage: status 400: chat not found")
	}}
	svc := &NotificationService{telegram: tg}
	assert.NotPanics(t, func() {
		svc.NotifySongRequest(&models.SongRequest{SongTitle: "Song"}, "kris")
	})
	assert.True(t, called)
}
