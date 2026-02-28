package models

import "database/sql"

type PlaylistTeam struct {
	ID         int            `db:"id"`
	PlaylistID int            `db:"playlist_id"`
	LeadID     int            `db:"lead_id"`
	MembersRaw sql.NullString `db:"members"`
	CreatedAt  sql.NullTime   `db:"createdAt"`
	UpdatedAt  sql.NullTime   `db:"updatedAt"`
}
