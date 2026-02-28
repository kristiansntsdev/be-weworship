package models

import "database/sql"

type Song struct {
	ID             int            `db:"id"`
	Title          string         `db:"title"`
	Artist         sql.NullString `db:"artist"`
	BaseChord      sql.NullString `db:"base_chord"`
	LyricsAndChord sql.NullString `db:"lyrics_and_chords"`
	CreatedAt      sql.NullTime   `db:"createdAt"`
	UpdatedAt      sql.NullTime   `db:"updatedAt"`
}

type Tag struct {
	ID          int            `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
}
