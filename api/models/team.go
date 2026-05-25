package models

import (
	"database/sql"
	"time"
)

type PlaylistTeam struct {
	ID          int            `db:"id"`
	PlaylistID  int            `db:"playlist_id"`
	LeadID      int            `db:"lead_id"`
	MembersRaw  sql.NullString `db:"members"`
	CoLeadsRaw  sql.NullString `db:"co_leads"`
	CreatedAt   sql.NullTime   `db:"createdAt"`
	UpdatedAt   sql.NullTime   `db:"updatedAt"`
}

// PlaylistMember represents a row in the playlist_members table.
// role is one of: 'singer' | 'musician' | 'md'
type PlaylistMember struct {
	ID         int       `db:"id"`
	PlaylistID int       `db:"playlist_id"`
	UserID     int       `db:"user_id"`
	Role       string    `db:"role"`
	JoinedAt   time.Time `db:"joined_at"`
}

// PlaylistMemberDetail joins playlist_members with the users table for API responses.
type PlaylistMemberDetail struct {
	UserID   int    `db:"user_id"`
	Username string `db:"username"`
	Email    string `db:"email"`
	Role     string `db:"role"`
}
