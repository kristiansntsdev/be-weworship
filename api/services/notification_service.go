package services

import (
	"context"
	"log"
	"time"

	"be-songbanks-v1/api/providers"
	"be-songbanks-v1/api/repositories"
)

// NotificationService orchestrates push notifications via the Expo Push API.
// Sends are synchronous so they complete before the Vercel handler returns.
type NotificationService struct {
	push *providers.ExpoPushProvider
	repo *repositories.NotificationRepository
}

func NewNotificationService(push *providers.ExpoPushProvider, repo *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{push: push, repo: repo}
}

// SaveDeviceToken upserts a device token for a user.
func (s *NotificationService) SaveDeviceToken(userID int, token, platform string) error {
	return s.repo.UpsertDeviceToken(userID, token, platform)
}

// RemoveDeviceToken deletes a specific token (called on logout).
func (s *NotificationService) RemoveDeviceToken(userID int, token string) error {
	return s.repo.DeleteDeviceToken(userID, token)
}

// pushCtx returns a context with a 15-second timeout for push API calls.
func pushCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

// NotifyNewSong sends a push notification to ALL registered devices and saves a
// broadcast inbox row (user_id = NULL) visible to every user.
func (s *NotificationService) NotifyNewSong(title string) {
	notifTitle := "🎵 New Song Available"
	msg := title + " is now available on WeWorship!"
	log.Printf("[notification] NotifyNewSong: saving broadcast inbox row for %q", title)
	if err := s.repo.SaveBroadcastNotification(notifTitle, msg, "new_song", `{"type":"new_song"}`); err != nil {
		log.Printf("[notification] NotifyNewSong: SaveBroadcastNotification failed: %v", err)
	}
	tokens, err := s.repo.GetAllTokens()
	if err != nil {
		log.Printf("[notification] NotifyNewSong: GetAllTokens failed: %v", err)
		return
	}
	log.Printf("[notification] NotifyNewSong: sending Expo push to %d token(s)", len(tokens))
	ctx, cancel := pushCtx()
	defer cancel()
	if err := s.push.Send(ctx, tokens, notifTitle, msg,
		map[string]string{"type": "new_song", "title": title},
	); err != nil {
		log.Printf("[notification] NotifyNewSong: Expo send failed: %v", err)
	}
}

// NotifyMemberLeft notifies the playlist owner when a member leaves, and saves inbox row.
func (s *NotificationService) NotifyMemberLeft(playlistName, memberName string, ownerID int) {
	if ownerID == 0 {
		return
	}
	notifTitle := "👋 Member Left"
	notifBody := memberName + " left your \"" + playlistName + "\" playlist."
	log.Printf("[notification] NotifyMemberLeft: saving inbox row for ownerID=%d", ownerID)
	if err := s.repo.SaveNotification(ownerID, notifTitle, notifBody, "playlist_update", `{"type":"playlist_update"}`); err != nil {
		log.Printf("[notification] NotifyMemberLeft: SaveNotification failed: %v", err)
	}
	tokens, err := s.repo.GetTokensByUserIDs([]int{ownerID})
	if err != nil {
		log.Printf("[notification] NotifyMemberLeft: GetTokens failed: %v", err)
		return
	}
	log.Printf("[notification] NotifyMemberLeft: sending Expo push to %d token(s) for ownerID=%d", len(tokens), ownerID)
	ctx, cancel := pushCtx()
	defer cancel()
	if err := s.push.Send(ctx, tokens, notifTitle, notifBody,
		map[string]string{"type": "playlist_update", "playlist": playlistName},
	); err != nil {
		log.Printf("[notification] NotifyMemberLeft: Expo send failed: %v", err)
	}
}

// NotifySongRequestUpdated notifies the requester on approval/rejection, and saves inbox row.
func (s *NotificationService) NotifySongRequestUpdated(songTitle, status string, requesterID int) {
	if requesterID == 0 {
		return
	}
	var notifTitle, notifBody string
	if status == "approved" {
		notifTitle = "✅ Song Request Approved"
		notifBody = "Your request for \"" + songTitle + "\" has been approved!"
	} else {
		notifTitle = "❌ Song Request Rejected"
		notifBody = "Your request for \"" + songTitle + "\" was not approved."
	}
	log.Printf("[notification] NotifySongRequestUpdated: saving inbox row for requesterID=%d status=%s", requesterID, status)
	if err := s.repo.SaveNotification(requesterID, notifTitle, notifBody, "system", `{"type":"system"}`); err != nil {
		log.Printf("[notification] NotifySongRequestUpdated: SaveNotification failed: %v", err)
	}
	tokens, err := s.repo.GetTokensByUserIDs([]int{requesterID})
	if err != nil {
		log.Printf("[notification] NotifySongRequestUpdated: GetTokens failed: %v", err)
		return
	}
	log.Printf("[notification] NotifySongRequestUpdated: sending Expo push to %d token(s) for requesterID=%d", len(tokens), requesterID)
	ctx, cancel := pushCtx()
	defer cancel()
	if err := s.push.Send(ctx, tokens, notifTitle, notifBody,
		map[string]string{"type": "system", "song_title": songTitle, "status": status},
	); err != nil {
		log.Printf("[notification] NotifySongRequestUpdated: Expo send failed: %v", err)
	}
}

// NotifyPlaylistRenamed notifies members that the playlist was renamed by the owner.
func (s *NotificationService) NotifyPlaylistRenamed(oldName, newName string, memberIDs []int) {
	if len(memberIDs) == 0 {
		return
	}
	notifTitle := "✏️ Playlist Renamed"
	notifBody := "\"" + oldName + "\" has been renamed to \"" + newName + "\"."
	log.Printf("[notification] NotifyPlaylistRenamed: saving inbox rows for %d member(s)", len(memberIDs))
	for _, uid := range memberIDs {
		if err := s.repo.SaveNotification(uid, notifTitle, notifBody, "playlist_update", `{"type":"playlist_update"}`); err != nil {
			log.Printf("[notification] NotifyPlaylistRenamed: SaveNotification uid=%d failed: %v", uid, err)
		}
	}
	tokens, err := s.repo.GetTokensByUserIDs(memberIDs)
	if err != nil {
		log.Printf("[notification] NotifyPlaylistRenamed: GetTokensByUserIDs failed: %v", err)
		return
	}
	log.Printf("[notification] NotifyPlaylistRenamed: sending Expo push to %d token(s)", len(tokens))
	ctx, cancel := pushCtx()
	defer cancel()
	if err := s.push.Send(ctx, tokens, notifTitle, notifBody,
		map[string]string{"type": "playlist_update", "playlist": newName},
	); err != nil {
		log.Printf("[notification] NotifyPlaylistRenamed: Expo send failed: %v", err)
	}
}

// NotifyPlaylistUpdate notifies members of a playlist update, and saves inbox rows.
func (s *NotificationService) NotifyPlaylistUpdate(playlistName string, memberIDs []int) {
	if len(memberIDs) == 0 {
		return
	}
	notifTitle := "📋 Playlist Updated"
	notifBody := playlistName + " has been updated."
	log.Printf("[notification] NotifyPlaylistUpdate: saving inbox rows for %d member(s)", len(memberIDs))
	for _, uid := range memberIDs {
		if err := s.repo.SaveNotification(uid, notifTitle, notifBody, "playlist_update", `{"type":"playlist_update"}`); err != nil {
			log.Printf("[notification] NotifyPlaylistUpdate: SaveNotification uid=%d failed: %v", uid, err)
		}
	}
	tokens, err := s.repo.GetTokensByUserIDs(memberIDs)
	if err != nil {
		log.Printf("[notification] NotifyPlaylistUpdate: GetTokensByUserIDs failed: %v", err)
		return
	}
	log.Printf("[notification] NotifyPlaylistUpdate: sending Expo push to %d token(s)", len(tokens))
	ctx, cancel := pushCtx()
	defer cancel()
	if err := s.push.Send(ctx, tokens, notifTitle, notifBody,
		map[string]string{"type": "playlist_update", "playlist": playlistName},
	); err != nil {
		log.Printf("[notification] NotifyPlaylistUpdate: Expo send failed: %v", err)
	}
}

// ── Inbox query methods ───────────────────────────────────────────────────────

func (s *NotificationService) GetNotifications(userID, page, limit int) ([]repositories.NotificationRow, error) {
	return s.repo.ListByUserID(userID, page, limit)
}

func (s *NotificationService) MarkAsRead(id, userID int) error {
	return s.repo.MarkRead(id, userID)
}

func (s *NotificationService) GetUnreadCount(userID int) (int, error) {
	return s.repo.CountUnread(userID)
}

// PushStatus returns push provider info for the debug endpoint.
func (s *NotificationService) FCMStatus() (enabled bool, projectID string) {
	return true, "expo-push"
}
