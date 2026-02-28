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
	return &PlaylistRepository{db: db}
}

func (r *PlaylistRepository) NameExistsForUser(userID int, name string) (bool, error) {
	var count int
	if err := r.db.Get(&count, `SELECT COUNT(*) FROM playlists WHERE user_id=? AND LOWER(playlist_name)=LOWER(?)`, userID, name); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PlaylistRepository) Create(userID int, name string, songs []int) (int, error) {
	songsJSON, _ := json.Marshal(songs)
	res, err := r.db.Exec(`INSERT INTO playlists (playlist_name,user_id,songs,is_shared,is_locked,createdAt,updatedAt) VALUES (?,?,?,0,0,NOW(),NOW())`, name, userID, string(songsJSON))
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (r *PlaylistRepository) CountAccessible(userID int) (int, error) {
	var total int
	query := `SELECT COUNT(DISTINCT p.id)
	FROM playlists p
	LEFT JOIN playlist_teams pt ON p.id = pt.playlist_id
	WHERE p.user_id = ? OR JSON_CONTAINS(COALESCE(pt.members,'[]'), ?)`
	err := r.db.Get(&total, query, userID, fmt.Sprintf("%d", userID))
	return total, err
}

func (r *PlaylistRepository) ListAccessible(userID, page, limit int) ([]models.PlaylistListRow, error) {
	offset := (page - 1) * limit
	query := `SELECT DISTINCT p.id,p.playlist_name,p.user_id,p.songs,p.playlist_notes,p.createdAt,p.updatedAt,
	CASE WHEN p.user_id = ? THEN 'owner' WHEN pt.lead_id = ? THEN 'leader' ELSE 'member' END AS access_type
	FROM playlists p
	LEFT JOIN playlist_teams pt ON p.id = pt.playlist_id
	WHERE p.user_id = ? OR JSON_CONTAINS(COALESCE(pt.members,'[]'), ?)
	ORDER BY p.createdAt DESC LIMIT ? OFFSET ?`
	rows := []models.PlaylistListRow{}
	err := r.db.Select(&rows, query, userID, userID, userID, fmt.Sprintf("%d", userID), limit, offset)
	return rows, err
}

func (r *PlaylistRepository) GetByID(playlistID int) (*models.Playlist, error) {
	var pl models.Playlist
	err := r.db.Get(&pl, `SELECT id,playlist_name,sharable_link,share_token,user_id,playlist_team_id,is_shared,is_locked,createdAt,updatedAt,songs,playlist_notes FROM playlists WHERE id=?`, playlistID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pl, nil
}

func (r *PlaylistRepository) UpdateName(playlistID, userID int, name string) (int64, error) {
	res, err := r.db.Exec(`UPDATE playlists SET playlist_name=?,updatedAt=NOW() WHERE id=? AND user_id=?`, name, playlistID, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PlaylistRepository) Delete(playlistID, userID int) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM playlists WHERE id=? AND user_id=?`, playlistID, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PlaylistRepository) UpdateShare(playlistID int, link, token string, teamID int64) error {
	_, err := r.db.Exec(`UPDATE playlists SET sharable_link=?,share_token=?,playlist_team_id=?,is_shared=1,updatedAt=NOW() WHERE id=?`, link, token, teamID, playlistID)
	return err
}

func (r *PlaylistRepository) FindByShareToken(shareToken string) (playlistID, ownerID int, teamID sql.NullInt64, err error) {
	err = r.db.QueryRow(`SELECT id,user_id,playlist_team_id FROM playlists WHERE share_token=? AND is_shared=1`, shareToken).Scan(&playlistID, &ownerID, &teamID)
	return
}

func (r *PlaylistRepository) SetTeamID(playlistID int, teamID int64) error {
	_, err := r.db.Exec(`UPDATE playlists SET playlist_team_id=?,updatedAt=NOW() WHERE id=?`, teamID, playlistID)
	return err
}

func (r *PlaylistRepository) ClearShareAndTeam(playlistID int) error {
	_, err := r.db.Exec(`UPDATE playlists SET sharable_link=NULL,share_token=NULL,is_shared=0,playlist_team_id=NULL,updatedAt=NOW() WHERE id=?`, playlistID)
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
	_, err := r.db.Exec(`UPDATE playlists SET songs=?,updatedAt=NOW() WHERE id=?`, string(buf), playlistID)
	return err
}

func (r *PlaylistRepository) ExistsAndOwner(playlistID int) (int, error) {
	var userID int
	err := r.db.Get(&userID, `SELECT user_id FROM playlists WHERE id=?`, playlistID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}
