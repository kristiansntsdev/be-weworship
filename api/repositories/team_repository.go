package repositories

import (
	"database/sql"
	"encoding/json"

	"be-songbanks-v1/api/models"
	"github.com/jmoiron/sqlx"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(playlistID, leadID int, members []int) (int64, error) {
	buf, _ := json.Marshal(members)
	var id int64
	err := r.db.QueryRow(r.db.Rebind(`INSERT INTO playlist_teams (playlist_id,lead_id,members,"createdAt","updatedAt") VALUES (?,?,?,NOW(),NOW()) RETURNING id`), playlistID, leadID, string(buf)).Scan(&id)
	return id, err
}

func (r *TeamRepository) GetByID(id int) (*models.PlaylistTeam, error) {
	var t models.PlaylistTeam
	err := r.db.Get(&t, r.db.Rebind(`SELECT id,playlist_id,lead_id,members,co_leads,"createdAt","updatedAt" FROM playlist_teams WHERE id=?`), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepository) FindByPlaylistID(playlistID int) (*models.PlaylistTeam, error) {
	var t models.PlaylistTeam
	err := r.db.Get(&t, r.db.Rebind(`SELECT id,playlist_id,lead_id,members,co_leads,"createdAt","updatedAt" FROM playlist_teams WHERE playlist_id=? LIMIT 1`), playlistID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepository) ListByLeadID(leadID int) ([]models.PlaylistTeam, error) {
	rows := []models.PlaylistTeam{}
	err := r.db.Select(&rows, r.db.Rebind(`SELECT id,playlist_id,lead_id,members,co_leads,"createdAt","updatedAt" FROM playlist_teams WHERE lead_id=? ORDER BY "createdAt" DESC`), leadID)
	return rows, err
}

func (r *TeamRepository) UpdateMembers(id int, members []int) error {
	buf, _ := json.Marshal(members)
	_, err := r.db.Exec(r.db.Rebind(`UPDATE playlist_teams SET members=?,"updatedAt"=NOW() WHERE id=?`), string(buf), id)
	return err
}

func (r *TeamRepository) UpdateCoLeads(id int, coLeads []int) error {
	buf, _ := json.Marshal(coLeads)
	_, err := r.db.Exec(r.db.Rebind(`UPDATE playlist_teams SET co_leads=?,"updatedAt"=NOW() WHERE id=?`), string(buf), id)
	return err
}

func (r *TeamRepository) Delete(id int) error {
	_, err := r.db.Exec(r.db.Rebind(`DELETE FROM playlist_teams WHERE id=?`), id)
	return err
}

// ── playlist_members ──────────────────────────────────────────────────────────

// ListMembers returns all members with their roles for a playlist, joined with user info.
func (r *TeamRepository) ListMembers(playlistID int) ([]models.PlaylistMemberDetail, error) {
	rows := []models.PlaylistMemberDetail{}
	err := r.db.Select(&rows, r.db.Rebind(`
		SELECT pm.user_id, u.username, u.email, pm.role
		FROM playlist_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.playlist_id = ?
		ORDER BY pm.joined_at ASC
	`), playlistID)
	return rows, err
}

// GetMember returns a single member row, or nil if not found.
func (r *TeamRepository) GetMember(playlistID, userID int) (*models.PlaylistMember, error) {
	var m models.PlaylistMember
	err := r.db.Get(&m, r.db.Rebind(`
		SELECT id, playlist_id, user_id, role, joined_at
		FROM playlist_members
		WHERE playlist_id = ? AND user_id = ?
	`), playlistID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CountByRole counts members with a given role in a playlist.
func (r *TeamRepository) CountByRole(playlistID int, role string) (int, error) {
	var count int
	err := r.db.QueryRow(r.db.Rebind(`
		SELECT COUNT(*) FROM playlist_members WHERE playlist_id = ? AND role = ?
	`), playlistID, role).Scan(&count)
	return count, err
}

// UpsertMember inserts or updates a member's role.
func (r *TeamRepository) UpsertMember(playlistID, userID int, role string) error {
	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO playlist_members (playlist_id, user_id, role, joined_at)
		VALUES (?, ?, ?, NOW())
		ON CONFLICT (playlist_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`), playlistID, userID, role)
	return err
}
