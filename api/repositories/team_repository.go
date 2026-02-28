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
	res, err := r.db.Exec(`INSERT INTO playlist_teams (playlist_id,lead_id,members,createdAt,updatedAt) VALUES (?,?,?,NOW(),NOW())`, playlistID, leadID, string(buf))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *TeamRepository) GetByID(id int) (*models.PlaylistTeam, error) {
	var t models.PlaylistTeam
	err := r.db.Get(&t, `SELECT id,playlist_id,lead_id,members,createdAt,updatedAt FROM playlist_teams WHERE id=?`, id)
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
	err := r.db.Get(&t, `SELECT id,playlist_id,lead_id,members,createdAt,updatedAt FROM playlist_teams WHERE playlist_id=? LIMIT 1`, playlistID)
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
	err := r.db.Select(&rows, `SELECT id,playlist_id,lead_id,members,createdAt,updatedAt FROM playlist_teams WHERE lead_id=? ORDER BY createdAt DESC`, leadID)
	return rows, err
}

func (r *TeamRepository) UpdateMembers(id int, members []int) error {
	buf, _ := json.Marshal(members)
	_, err := r.db.Exec(`UPDATE playlist_teams SET members=?,updatedAt=NOW() WHERE id=?`, string(buf), id)
	return err
}

func (r *TeamRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM playlist_teams WHERE id=?`, id)
	return err
}
