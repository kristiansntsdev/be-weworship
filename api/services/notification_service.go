package services

import (
	"context"
	"log"

	"be-songbanks-v1/api/providers"
	"be-songbanks-v1/api/repositories"
)

// NotificationService orchestrates push notifications via FCM.
// All notification sends are fire-and-forget (goroutines) so they
// never block the HTTP response.
type NotificationService struct {
	fcm  *providers.FCMProvider
	repo *repositories.NotificationRepository
}

func NewNotificationService(fcm *providers.FCMProvider, repo *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{fcm: fcm, repo: repo}
}

// SaveDeviceToken upserts a device token for a user.
func (s *NotificationService) SaveDeviceToken(userID int, token, platform string) error {
	return s.repo.UpsertDeviceToken(userID, token, platform)
}

// RemoveDeviceToken deletes a specific token (called on logout).
func (s *NotificationService) RemoveDeviceToken(userID int, token string) error {
	return s.repo.DeleteDeviceToken(userID, token)
}

// NotifyNewSong sends a push notification to all devices subscribed to the
// "new-songs" FCM topic AND saves a broadcast inbox row (user_id = NULL).
func (s *NotificationService) NotifyNewSong(title string) {
	if s.fcm == nil {
		return
	}
	msg := title + " is now available on WeWorship!"
	// Persist broadcast inbox row (visible to all users).
	if err := s.repo.SaveBroadcastNotification("🎵 New Song Available", msg, "new_song", `{"type":"new_song"}`); err != nil {
		log.Printf("[notification] SaveBroadcastNotification failed: %v", err)
	}
	go func() {
		ctx := context.Background()
		err := s.fcm.SendToTopic(ctx, "new-songs",
			"🎵 New Song Available",
			msg,
			map[string]string{"type": "new_song", "title": title},
		)
		if err != nil {
			log.Printf("[notification] NotifyNewSong failed: %v", err)
		}
	}()
}

// NotifyMemberLeft sends a push notification to the playlist owner when a
// member leaves their team, and saves a targeted inbox row for the owner.
func (s *NotificationService) NotifyMemberLeft(playlistName, memberName string, ownerID int) {
	if ownerID == 0 {
		return
	}
	notifTitle := "👋 Member Left"
	notifBody := memberName + " left your \"" + playlistName + "\" playlist."
	if err := s.repo.SaveNotification(ownerID, notifTitle, notifBody, "playlist_update", `{"type":"playlist_update"}`); err != nil {
		log.Printf("[notification] SaveNotification(member_left) failed: %v", err)
	}
	if s.fcm == nil {
		return
	}
	go func() {
		tokens, err := s.repo.GetTokensByUserIDs([]int{ownerID})
		if err != nil {
			log.Printf("[notification] NotifyMemberLeft GetTokens failed: %v", err)
			return
		}
		ctx := context.Background()
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, notifTitle, notifBody,
				map[string]string{"type": "playlist_update", "playlist": playlistName},
			); err != nil {
				log.Printf("[notification] NotifyMemberLeft SendToToken failed for token %s: %v", token, err)
			}
		}
	}()
}

// NotifySongRequestUpdated sends a push notification to the requester when
// their song request status changes to approved or rejected, and saves to inbox.
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
	if err := s.repo.SaveNotification(requesterID, notifTitle, notifBody, "system", `{"type":"system"}`); err != nil {
		log.Printf("[notification] SaveNotification(song_request) failed: %v", err)
	}
	if s.fcm == nil {
		return
	}
	go func() {
		tokens, err := s.repo.GetTokensByUserIDs([]int{requesterID})
		if err != nil {
			log.Printf("[notification] NotifySongRequestUpdated GetTokens failed: %v", err)
			return
		}
		ctx := context.Background()
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, notifTitle, notifBody,
				map[string]string{"type": "system", "song_title": songTitle, "status": status},
			); err != nil {
				log.Printf("[notification] NotifySongRequestUpdated SendToToken failed for token %s: %v", token, err)
			}
		}
	}()
}

// NotifyPlaylistUpdate sends a push notification to all devices belonging to
// the given member user IDs and saves a targeted inbox row for each member.
func (s *NotificationService) NotifyPlaylistUpdate(playlistName string, memberIDs []int) {
	if len(memberIDs) == 0 {
		return
	}
	notifTitle := "📋 Playlist Updated"
	notifBody := playlistName + " has been updated."
	for _, uid := range memberIDs {
		if err := s.repo.SaveNotification(uid, notifTitle, notifBody, "playlist_update", `{"type":"playlist_update"}`); err != nil {
			log.Printf("[notification] SaveNotification(playlist_update) uid=%d failed: %v", uid, err)
		}
	}
	if s.fcm == nil {
		return
	}
	go func() {
		tokens, err := s.repo.GetTokensByUserIDs(memberIDs)
		if err != nil {
			log.Printf("[notification] GetTokensByUserIDs failed: %v", err)
			return
		}
		ctx := context.Background()
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, notifTitle, notifBody,
				map[string]string{"type": "playlist_update", "playlist": playlistName},
			); err != nil {
				log.Printf("[notification] SendToToken failed for token %s: %v", token, err)
			}
		}
	}()
}

// ── Inbox query methods ───────────────────────────────────────────────────────

// GetNotifications returns paginated inbox items for a user (targeted + broadcasts).
func (s *NotificationService) GetNotifications(userID, page, limit int) ([]repositories.NotificationRow, error) {
	return s.repo.ListByUserID(userID, page, limit)
}

// MarkAsRead marks a targeted notification as read for the given user.
func (s *NotificationService) MarkAsRead(id, userID int) error {
	return s.repo.MarkRead(id, userID)
}

// GetUnreadCount returns the count of unread targeted notifications (broadcasts excluded).
func (s *NotificationService) GetUnreadCount(userID int) (int, error) {
	return s.repo.CountUnread(userID)
}
