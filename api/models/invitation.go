package models

import "time"

// PlaylistInvitation represents a row in the playlist_invitations table.
// status: "pending" | "accepted" | "declined"
type PlaylistInvitation struct {
	ID         int       `db:"id"`
	PlaylistID int       `db:"playlist_id"`
	InviterID  int       `db:"inviter_id"`
	InviteeID  int       `db:"invitee_id"`
	Status     string    `db:"status"`
	CreatedAt  time.Time `db:"createdAt"`
	UpdatedAt  time.Time `db:"updatedAt"`
}
