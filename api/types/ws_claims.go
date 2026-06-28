package types

import "github.com/golang-jwt/jwt/v5"

// WSClaims is a short-lived token for realtime WebSocket connections.
type WSClaims struct {
	UserID           int    `json:"userId"`
	PlaylistID       int    `json:"playlistId"`
	MemberRole       string `json:"memberRole"` // singer|musician|md|guest
	Username         string `json:"username,omitempty"`
	CanPublishScreen bool   `json:"canPublishScreen"`
	CanPublishAudio  bool   `json:"canPublishAudio"`
	jwt.RegisteredClaims
}
