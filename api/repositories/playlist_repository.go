package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/utils"
	"github.com/jmoiron/sqlx"
)

type PlaylistRepository struct {
	db *sqlx.DB
}

func NewPlaylistRepository(db *sqlx.DB) *PlaylistRepository {
	db.Exec(`CREATE TABLE IF NOT EXISTS playlist_share_events (
		id          SERIAL PRIMARY KEY,
		playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
		"createdAt" TIMESTAMP NOT NULL DEFAULT NOW()
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_pse_created ON playlist_share_events ("createdAt")`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_pse_playlist ON playlist_share_events (playlist_id)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS playlist_song_keys (
		playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
		song_id     INTEGER NOT NULL,
		base_chord  VARCHAR(10) NOT NULL,
		PRIMARY KEY (playlist_id, song_id)
	)`)
	return &PlaylistRepository{db: db}
}

// RecordShareEvent records a share access (join via link) for weekly stats.
func (r *PlaylistRepository) RecordShareEvent(playlistID int) {
	r.db.Exec(r.db.Rebind(`INSERT INTO playlist_share_events (playlist_id) VALUES (?)`), playlistID)
}

// CountShareEventsThisWeek returns the total join-via-link events since Monday 00:00 of the current week.
func (r *PlaylistRepository) CountShareEventsThisWeek() (int, error) {
	var count int
	err := r.db.Get(&count, `
		SELECT COUNT(*) FROM playlist_share_events
		WHERE "createdAt" >= DATE_TRUNC('week', NOW())
	`)
	return count, err
}

func (r *PlaylistRepository) CountShareable() (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM playlists WHERE is_shared=1`)
	return count, err
}

func (r *PlaylistRepository) NameExistsForUser(userID int, name string) (bool, error) {
	var count int
	if err := r.db.Get(&count, r.db.Rebind(`SELECT COUNT(*) FROM playlists WHERE user_id=? AND LOWER(playlist_name)=LOWER(?)`), userID, name); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PlaylistRepository) Create(userID int, name string, songs []int) (int, error) {
	songsJSON, _ := json.Marshal(songs)
	var id int
	err := r.db.QueryRow(r.db.Rebind(`INSERT INTO playlists (playlist_name,user_id,songs,is_shared,is_locked,"createdAt","updatedAt") VALUES (?,?,?,0,0,NOW(),NOW()) RETURNING id`), name, userID, string(songsJSON)).Scan(&id)
	return id, err
}

func (r *PlaylistRepository) CountAccessible(userID int) (int, error) {
	var total int
	query := r.db.Rebind(`SELECT COUNT(DISTINCT p.id)
	FROM playlists p
	LEFT JOIN playlist_teams pt ON p.id = pt.playlist_id
	WHERE p.user_id = ? OR COALESCE(pt.members,'[]')::jsonb @> ?::jsonb`)
	err := r.db.Get(&total, query, userID, fmt.Sprintf("[%d]", userID))
	return total, err
}

func (r *PlaylistRepository) ListAccessible(userID, page, limit int) ([]models.PlaylistListRow, error) {
	offset := (page - 1) * limit
	query := r.db.Rebind(`SELECT DISTINCT p.id,p.playlist_name,p.user_id,p.songs,p.playlist_notes,p."createdAt",p."updatedAt",
	CASE WHEN p.user_id = ? THEN 'owner' WHEN pt.lead_id = ? THEN 'leader' ELSE 'member' END AS access_type
	FROM playlists p
	LEFT JOIN playlist_teams pt ON p.id = pt.playlist_id
	WHERE p.user_id = ? OR COALESCE(pt.members,'[]')::jsonb @> ?::jsonb
	ORDER BY p."createdAt" DESC LIMIT ? OFFSET ?`)
	rows := []models.PlaylistListRow{}
	err := r.db.Select(&rows, query, userID, userID, userID, fmt.Sprintf("[%d]", userID), limit, offset)
	return rows, err
}

func (r *PlaylistRepository) GetByID(playlistID int) (*models.Playlist, error) {
	var pl models.Playlist
	err := r.db.Get(&pl, r.db.Rebind(`SELECT id,playlist_name,sharable_link,share_token,user_id,playlist_team_id,is_shared,is_locked,"createdAt","updatedAt",songs,playlist_notes FROM playlists WHERE id=?`), playlistID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pl, nil
}

func (r *PlaylistRepository) UpdateName(playlistID, userID int, name string) (int64, error) {
	res, err := r.db.Exec(r.db.Rebind(`UPDATE playlists SET playlist_name=?,"updatedAt"=NOW() WHERE id=? AND user_id=?`), name, playlistID, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PlaylistRepository) Delete(playlistID, userID int) (int64, error) {
	res, err := r.db.Exec(r.db.Rebind(`DELETE FROM playlists WHERE id=? AND user_id=?`), playlistID, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PlaylistRepository) UpdateShare(playlistID int, link, token string, teamID int64) error {
	_, err := r.db.Exec(r.db.Rebind(`UPDATE playlists SET sharable_link=?,share_token=?,playlist_team_id=?,is_shared=1,"updatedAt"=NOW() WHERE id=?`), link, token, teamID, playlistID)
	return err
}

func (r *PlaylistRepository) FindByShareToken(shareToken string) (playlistID, ownerID int, teamID sql.NullInt64, err error) {
	err = r.db.QueryRow(r.db.Rebind(`SELECT id,user_id,playlist_team_id FROM playlists WHERE share_token=? AND is_shared=1`), shareToken).Scan(&playlistID, &ownerID, &teamID)
	return
}

type PlaylistPreview struct {
	Name      string `db:"playlist_name"`
	OwnerName string `db:"owner_name"`
	SongsRaw  string `db:"songs"`
}

func (r *PlaylistRepository) GetPreviewByShareToken(shareToken string) (*PlaylistPreview, error) {
	var p PlaylistPreview
	err := r.db.QueryRowx(r.db.Rebind(`
		SELECT p.playlist_name, u.name AS owner_name, COALESCE(p.songs,'') AS songs
		FROM playlists p
		JOIN users u ON u.id = p.user_id
		WHERE p.share_token=? AND p.is_shared=1
	`), shareToken).StructScan(&p)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlaylistRepository) SetTeamID(playlistID int, teamID int64) error {
	_, err := r.db.Exec(r.db.Rebind(`UPDATE playlists SET playlist_team_id=?,"updatedAt"=NOW() WHERE id=?`), teamID, playlistID)
	return err
}

func (r *PlaylistRepository) ClearShareAndTeam(playlistID int) error {
	_, err := r.db.Exec(r.db.Rebind(`UPDATE playlists SET sharable_link=NULL,share_token=NULL,is_shared=0,playlist_team_id=NULL,"updatedAt"=NOW() WHERE id=?`), playlistID)
	return err
}

func (r *PlaylistRepository) CanManage(playlistID, userID int) (bool, error) {
	pl, err := r.GetByID(playlistID)
	if err != nil || pl == nil {
		return false, err
	}
	if pl.UserID == userID {
		return true, nil
	}
	tm := NewTeamRepository(r.db)
	team, err := tm.FindByPlaylistID(playlistID)
	if err != nil || team == nil {
		return false, err
	}
	members := utils.ParseIntSlice(team.MembersRaw.String)
	return team.LeadID == userID || utils.ContainsInt(members, userID), nil
}

func (r *PlaylistRepository) SetSongs(playlistID int, songs []int) error {
	buf, _ := json.Marshal(songs)
	_, err := r.db.Exec(r.db.Rebind(`UPDATE playlists SET songs=?,"updatedAt"=NOW() WHERE id=?`), string(buf), playlistID)
	return err
}

func (r *PlaylistRepository) ExistsAndOwner(playlistID int) (int, error) {
	var userID int
	err := r.db.Get(&userID, r.db.Rebind(`SELECT user_id FROM playlists WHERE id=?`), playlistID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (r *PlaylistRepository) SetSongKey(playlistID, songID int, baseChord string) error {
	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO playlist_song_keys (playlist_id, song_id, base_chord)
		VALUES (?, ?, ?)
		ON CONFLICT (playlist_id, song_id) DO UPDATE SET base_chord = EXCLUDED.base_chord
	`), playlistID, songID, baseChord)
	return err
}

func (r *PlaylistRepository) GetSongKeys(playlistID int) (map[int]string, error) {
	rows, err := r.db.Query(r.db.Rebind(`SELECT song_id, base_chord FROM playlist_song_keys WHERE playlist_id = ?`), playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int]string{}
	for rows.Next() {
		var songID int
		var baseChord string
		if err := rows.Scan(&songID, &baseChord); err != nil {
			return nil, err
		}
		result[songID] = baseChord
	}
	return result, rows.Err()
}

func (r *PlaylistRepository) DeleteSongKey(playlistID, songID int) error {
	_, err := r.db.Exec(r.db.Rebind(`DELETE FROM playlist_song_keys WHERE playlist_id = ? AND song_id = ?`), playlistID, songID)
	return err
}
