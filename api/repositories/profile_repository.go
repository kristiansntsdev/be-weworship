package repositories

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// ProfileRepository reads the public face of users: what other users may see when they search for someone.
// It never selects email, role or address fields.
type ProfileRepository struct {
	db *sqlx.DB
}

func NewProfileRepository(db *sqlx.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// ProfileRow is a user as other users see them. DisplayName falls back to the username.
type ProfileRow struct {
	ID              int            `db:"id"`
	Username        string         `db:"username"`
	DisplayName     string         `db:"display_name"`
	AvatarURL       sql.NullString `db:"avatar_url"`
	Verified        bool           `db:"verified"`
	PublicPlaylists int            `db:"public_playlists"`
}

// PublicPlaylistRow is one public playlist on a profile.
type PublicPlaylistRow struct {
	ID           int            `db:"id"`
	PlaylistName string         `db:"playlist_name"`
	SongsRaw     sql.NullString `db:"songs"`
	UpdatedAt    time.Time      `db:"updatedAt"`
}

const profileColumns = `u.id, u.username, COALESCE(NULLIF(TRIM(ud.full_name), ''), u.username) AS display_name,
	u.avatar_url, u.verified,
	(SELECT COUNT(*) FROM playlists p WHERE p.user_id = u.id AND p.is_public = 1)::int AS public_playlists`

// Search finds active users whose username or display name contains q: exact username first, then username
// prefix, then verified accounts, then alphabetical.
func (r *ProfileRepository) Search(q string, limit int) ([]ProfileRow, error) {
	rows := []ProfileRow{}
	like := containsPattern(q)
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT `+profileColumns+`
		FROM users u
		LEFT JOIN users_detail ud ON ud.user_id = u.id
		WHERE u.status = 'active' AND (u.username ILIKE ? ESCAPE '\' OR ud.full_name ILIKE ? ESCAPE '\')
		ORDER BY (LOWER(u.username) = LOWER(?)) DESC, (u.username ILIKE ? ESCAPE '\') DESC, u.verified DESC, u.username ASC
		LIMIT ?`), like, like, q, escapeLikePattern(q)+"%", limit)
	return rows, err
}

// FindByUsername returns the active user with this username (case-insensitive), or nil.
func (r *ProfileRepository) FindByUsername(username string) (*ProfileRow, error) {
	var row ProfileRow
	err := r.db.Get(&row, r.db.Rebind(`
		SELECT `+profileColumns+`
		FROM users u
		LEFT JOIN users_detail ud ON ud.user_id = u.id
		WHERE u.status = 'active' AND LOWER(u.username) = LOWER(?)`), username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// PublicPlaylists lists the public playlists a user owns, most recently updated first.
func (r *ProfileRepository) PublicPlaylists(userID int) ([]PublicPlaylistRow, error) {
	rows := []PublicPlaylistRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT id, playlist_name, songs, "updatedAt"
		FROM playlists
		WHERE user_id = ? AND is_public = 1
		ORDER BY "updatedAt" DESC, id DESC`), userID)
	return rows, err
}
