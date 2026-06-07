package models

import (
	"database/sql"
	"time"
)

type Song struct {
	ID              int            `db:"id"`
	Slug            sql.NullString `db:"slug"`
	Title           string         `db:"title"`
	Artist          sql.NullString `db:"artist"`
	BaseChord       sql.NullString `db:"base_chord"`
	Bpm             sql.NullInt64  `db:"bpm"`
	LyricsAndChord  sql.NullString `db:"lyrics_and_chords"`
	PlainLyrics     sql.NullString `db:"plain_lyrics"`
	ExternalLinks   sql.NullString `db:"external_links"`
	DmcaTakedown    bool           `db:"dmca_takedown"`
	DmcaStatusNotes sql.NullString `db:"dmca_status_notes"`
	CreatedBy       sql.NullInt64  `db:"created_by"`
	CreatedAt       sql.NullTime   `db:"createdAt"`
	UpdatedAt       sql.NullTime   `db:"updatedAt"`
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
	LyricsType    string    `db:"lyrics_type"   json:"lyrics_type"`
	Lyrics        string    `db:"lyrics"        json:"lyrics"`
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
	LyricsType    string         `db:"lyrics_type"`
	Lyrics        string         `db:"lyrics"`
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
		LyricsType:    r.LyricsType,
		Lyrics:        r.Lyrics,
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

type SongReport struct {
	ID          int       `db:"id"           json:"id"`
	UserID      int       `db:"user_id"      json:"user_id"`
	SongID      int       `db:"song_id"      json:"song_id"`
	SongTitle   string    `db:"song_title"   json:"song_title"`
	ReportType  string    `db:"report_type"  json:"report_type"`
	Description string    `db:"description"  json:"description"`
	EvidenceURL string    `db:"evidence_url" json:"evidence_url"`
	Status      string    `db:"status"       json:"status"`
	AdminNotes  *string   `db:"admin_notes"  json:"admin_notes,omitempty"`
	CreatedAt   time.Time `db:"createdAt"    json:"createdAt"`
	UpdatedAt   time.Time `db:"updatedAt"    json:"updatedAt"`
}

// SongReportRow is used for scanning report rows with nullable fields.
type SongReportRow struct {
	ID          int            `db:"id"`
	UserID      int            `db:"user_id"`
	SongID      int            `db:"song_id"`
	SongTitle   string         `db:"song_title"`
	ReportType  string         `db:"report_type"`
	Description string         `db:"description"`
	EvidenceURL string         `db:"evidence_url"`
	Status      string         `db:"status"`
	AdminNotes  sql.NullString `db:"admin_notes"`
	CreatedAt   sql.NullTime   `db:"createdAt"`
	UpdatedAt   sql.NullTime   `db:"updatedAt"`
}

// ToSongReport converts a row to a proper SongReport with clean JSON serialization.
func (r *SongReportRow) ToSongReport() *SongReport {
	report := &SongReport{
		ID:          r.ID,
		UserID:      r.UserID,
		SongID:      r.SongID,
		SongTitle:   r.SongTitle,
		ReportType:  r.ReportType,
		Description: r.Description,
		EvidenceURL: r.EvidenceURL,
		Status:      r.Status,
	}
	if r.AdminNotes.Valid {
		report.AdminNotes = &r.AdminNotes.String
	}
	if r.CreatedAt.Valid {
		report.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		report.UpdatedAt = r.UpdatedAt.Time
	}
	return report
}
