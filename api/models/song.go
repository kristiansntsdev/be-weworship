package models

import (
	"database/sql"
	"time"
)

type Song struct {
	ID               int            `db:"id"`
	Slug             sql.NullString `db:"slug"`
	Title            string         `db:"title"`
	Artist           sql.NullString `db:"artist"`
	BaseChord        sql.NullString `db:"base_chord"`
	Bpm              sql.NullInt64  `db:"bpm"`
	LyricsAndChord   sql.NullString `db:"lyrics_and_chords"`
	ExternalLinks    sql.NullString `db:"external_links"`
	DmcaTakedown     bool           `db:"dmca_takedown"`
	DmcaStatusNotes  sql.NullString `db:"dmca_status_notes"`
	CreatedBy        sql.NullInt64  `db:"created_by"`
	CreatedAt        sql.NullTime   `db:"createdAt"`
	UpdatedAt        sql.NullTime   `db:"updatedAt"`
}

type Tag struct {
	ID          int            `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
}

type SongRequest struct {
	ID            int       `db:"id"            json:"id"`
	UserID        int       `db:"user_id"       json:"user_id"`
	SongTitle     string    `db:"song_title"    json:"song_title"`
	ReferenceLink string    `db:"reference_link" json:"reference_link"`
	Status        string    `db:"status"        json:"status"`
	AdminNotes    *string   `db:"admin_notes"   json:"admin_notes,omitempty"`
	CreatedAt     time.Time `db:"createdAt"     json:"createdAt"`
	UpdatedAt     time.Time `db:"updatedAt"     json:"updatedAt"`
}

// SongRequestRow is used for scanning from DB with nullable fields
type SongRequestRow struct {
	ID            int            `db:"id"`
	UserID        int            `db:"user_id"`
	SongTitle     string         `db:"song_title"`
	ReferenceLink string         `db:"reference_link"`
	Status        string         `db:"status"`
	AdminNotes    sql.NullString `db:"admin_notes"`
	CreatedAt     sql.NullTime   `db:"createdAt"`
	UpdatedAt     sql.NullTime   `db:"updatedAt"`
}

// ToSongRequest converts a row to a proper SongRequest with clean JSON serialization
func (r *SongRequestRow) ToSongRequest() *SongRequest {
	sr := &SongRequest{
		ID:            r.ID,
		UserID:        r.UserID,
		SongTitle:     r.SongTitle,
		ReferenceLink: r.ReferenceLink,
		Status:        r.Status,
	}
	if r.AdminNotes.Valid {
		sr.AdminNotes = &r.AdminNotes.String
	}
	if r.CreatedAt.Valid {
		sr.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		sr.UpdatedAt = r.UpdatedAt.Time
	}
	return sr
}
