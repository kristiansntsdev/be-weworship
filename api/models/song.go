package models

import "database/sql"

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
