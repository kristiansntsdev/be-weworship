package repositories

import (
	"github.com/jmoiron/sqlx"
)

// MixRepository stores each user's generated mixes in user_mixes (see migrations/add_user_mixes.sql).
type MixRepository struct {
	db *sqlx.DB
}

func NewMixRepository(db *sqlx.DB) *MixRepository {
	return &MixRepository{db: db}
}

// MixRow is one stored mix. Key is mix-1..mix-3, supermix or mood-<id>; SongsJSON is a [{id,title,artist,key}] snapshot.
type MixRow struct {
	Key        string  `db:"mix_key"`
	Title      string  `db:"title"`
	Subtitle   string  `db:"subtitle"`
	SongsJSON  string  `db:"songs"`
	AgeSeconds float64 `db:"age_seconds"` // read only: time since the row was generated, by the database clock
}

// ListForUser returns all of the user's stored mixes, ordered by key.
func (r *MixRepository) ListForUser(userID int) ([]MixRow, error) {
	rows := []MixRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT mix_key, title, subtitle, songs, EXTRACT(EPOCH FROM (NOW() - "generatedAt"))::float8 AS age_seconds
		FROM user_mixes
		WHERE user_id = ?
		ORDER BY mix_key`), userID)
	return rows, err
}

const upsertMixSQL = `
	INSERT INTO user_mixes (user_id, mix_key, title, subtitle, songs, "generatedAt")
	VALUES (?, ?, ?, ?, ?, NOW())
	ON CONFLICT (user_id, mix_key) DO UPDATE
	SET title = EXCLUDED.title, subtitle = EXCLUDED.subtitle, songs = EXCLUDED.songs, "generatedAt" = EXCLUDED."generatedAt"`

// ReplacePersonal swaps the user's personal mixes (every key except mood-*) for the given set in one transaction.
// Upserting rather than delete-then-insert keeps two concurrent rebuilds from failing on the primary key.
func (r *MixRepository) ReplacePersonal(userID int, mixes []MixRow) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op once committed

	keys := make([]string, len(mixes))
	for i, m := range mixes {
		keys[i] = m.Key
	}
	if _, err := tx.Exec(tx.Rebind(`DELETE FROM user_mixes WHERE user_id = ? AND mix_key NOT LIKE 'mood-%' AND mix_key <> ALL(?)`), userID, keys); err != nil {
		return err
	}
	for _, m := range mixes {
		if _, err := tx.Exec(tx.Rebind(upsertMixSQL), userID, m.Key, m.Title, m.Subtitle, m.SongsJSON); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Upsert stores one mix (used for mood mixes, which are rebuilt one at a time).
func (r *MixRepository) Upsert(userID int, m MixRow) error {
	_, err := r.db.Exec(r.db.Rebind(upsertMixSQL), userID, m.Key, m.Title, m.Subtitle, m.SongsJSON)
	return err
}
