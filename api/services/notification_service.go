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
// "new-songs" FCM topic. Triggered when a song is created OR its
// lyrics_and_chords are first set (ChordPro-ready).
func (s *NotificationService) NotifyNewSong(title string) {
	if s.fcm == nil {
		return
	}
	go func() {
		ctx := context.Background()
		err := s.fcm.SendToTopic(ctx, "new-songs",
			"🎵 New Song Available",
			title+" is now available on WeWorship!",
			map[string]string{"type": "new_song", "title": title},
		)
		if err != nil {
			log.Printf("[notification] NotifyNewSong failed: %v", err)
		}
	}()
}

// NotifyMemberLeft sends a push notification to the playlist owner when a
// member leaves their team.
func (s *NotificationService) NotifyMemberLeft(playlistName, memberName string, ownerID int) {
	if s.fcm == nil || ownerID == 0 {
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
			if err := s.fcm.SendToToken(ctx, token,
				"👋 Member Left",
				memberName+" left your \""+playlistName+"\" playlist.",
				map[string]string{"type": "playlist_update", "playlist": playlistName},
			); err != nil {
				log.Printf("[notification] NotifyMemberLeft SendToToken failed for token %s: %v", token, err)
			}
		}
	}()
}

// NotifySongRequestUpdated sends a push notification to the requester when
// their song request status changes to approved or rejected.
func (s *NotificationService) NotifySongRequestUpdated(songTitle, status string, requesterID int) {
	if s.fcm == nil || requesterID == 0 {
		return
	}
	go func() {
		tokens, err := s.repo.GetTokensByUserIDs([]int{requesterID})
		if err != nil {
			log.Printf("[notification] NotifySongRequestUpdated GetTokens failed: %v", err)
			return
		}
		var title, body string
		if status == "approved" {
			title = "✅ Song Request Approved"
			body = "Your request for \"" + songTitle + "\" has been approved!"
		} else {
			title = "❌ Song Request Rejected"
			body = "Your request for \"" + songTitle + "\" was not approved."
		}
		ctx := context.Background()
		for _, token := range tokens {
			if err := s.fcm.SendToToken(ctx, token, title, body,
				map[string]string{"type": "system", "song_title": songTitle, "status": status},
			); err != nil {
				log.Printf("[notification] NotifySongRequestUpdated SendToToken failed for token %s: %v", token, err)
			}
		}
	}()
}

// NotifyPlaylistUpdate sends a push notification to all devices belonging to
// the given member user IDs. Used when a playlist is updated or someone joins.
func (s *NotificationService) NotifyPlaylistUpdate(playlistName string, memberIDs []int) {
	if s.fcm == nil || len(memberIDs) == 0 {
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
			if err := s.fcm.SendToToken(ctx, token,
				"📋 Playlist Updated",
				playlistName+" has been updated.",
				map[string]string{"type": "playlist_update", "playlist": playlistName},
			); err != nil {
				log.Printf("[notification] SendToToken failed for token %s: %v", token, err)
			}
		}
	}()
}
