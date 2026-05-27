package repositories

import (
	"database/sql"

	"be-songbanks-v1/api/models"
	"github.com/jmoiron/sqlx"
)

type InvitationRepository struct {
	db *sqlx.DB
}

func NewInvitationRepository(db *sqlx.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// CreateInvitation inserts a new pending invitation and returns its id.
func (r *InvitationRepository) CreateInvitation(playlistID, inviterID, inviteeID int) (int, error) {
	var id int
	err := r.db.QueryRow(
		`INSERT INTO playlist_invitations (playlist_id, inviter_id, invitee_id)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		playlistID, inviterID, inviteeID,
	).Scan(&id)
	return id, err
}

// FindPendingByID returns the invitation only if it exists and is still pending.
func (r *InvitationRepository) FindPendingByID(id int) (*models.PlaylistInvitation, error) {
	var inv models.PlaylistInvitation
	err := r.db.Get(&inv,
		`SELECT id, playlist_id, inviter_id, invitee_id, status, "createdAt", "updatedAt"
		 FROM playlist_invitations
		 WHERE id = $1 AND status = 'pending'`,
		id,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &inv, err
}

// FindPendingByPlaylistAndInvitee returns an existing pending invite for the given pair, or nil.
func (r *InvitationRepository) FindPendingByPlaylistAndInvitee(playlistID, inviteeID int) (*models.PlaylistInvitation, error) {
	var inv models.PlaylistInvitation
	err := r.db.Get(&inv,
		`SELECT id, playlist_id, inviter_id, invitee_id, status, "createdAt", "updatedAt"
		 FROM playlist_invitations
		 WHERE playlist_id = $1 AND invitee_id = $2 AND status = 'pending'
		 LIMIT 1`,
		playlistID, inviteeID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &inv, err
}

// UpdateStatus sets status and updatedAt for a given invitation id.
func (r *InvitationRepository) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec(
		`UPDATE playlist_invitations SET status = $1, "updatedAt" = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}
