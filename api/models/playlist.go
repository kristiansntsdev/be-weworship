package models

import (
	"database/sql"
	"time"
)

type Playlist struct {
	ID               int            `db:"id"`
	PlaylistName     string         `db:"playlist_name"`
	ShareableURL     sql.NullString `db:"sharable_link"`
	ShareToken       sql.NullString `db:"share_token"`
	UserID           int            `db:"user_id"`
	PlaylistTeamID   sql.NullInt64  `db:"playlist_team_id"`
	IsShared         bool           `db:"is_shared"`
	IsLocked         bool           `db:"is_locked"`
	CreatedAt        time.Time      `db:"createdAt"`
	UpdatedAt        time.Time      `db:"updatedAt"`
	SongsRaw         sql.NullString `db:"songs"`
	PlaylistNotesRaw sql.NullString `db:"playlist_notes"`
}

type PlaylistListRow struct {
	ID           int            `db:"id"`
	PlaylistName string         `db:"playlist_name"`
	UserID       int            `db:"user_id"`
	SongsRaw     sql.NullString `db:"songs"`
	NotesRaw     sql.NullString `db:"playlist_notes"`
	CreatedAt    sql.NullTime   `db:"createdAt"`
	UpdatedAt    sql.NullTime   `db:"updatedAt"`
	AccessType   string         `db:"access_type"`
}
