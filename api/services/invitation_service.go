package services

import (
	"database/sql"
	"fmt"
	"strings"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

// InvitationService handles playlist invite flows: send, accept, decline, and user search.
type InvitationService struct {
	invitations repositories.InvitationRepoIface
	playlists   repositories.PlaylistRepoIface
	teams       repositories.TeamRepoIface
	users       repositories.UserRepoIface
	notif       *NotificationService
}

func NewInvitationService(
	inv *repositories.InvitationRepository,
	pl *repositories.PlaylistRepository,
	t *repositories.TeamRepository,
	u *repositories.UserRepository,
	n *NotificationService,
) *InvitationService {
	return &InvitationService{invitations: inv, playlists: pl, teams: t, users: u, notif: n}
}

// SearchUsers returns up to 10 users whose username or email contains the query,
// excluding the requesting user. Returns empty slice for queries shorter than 2 chars.
func (s *InvitationService) SearchUsers(query string, requestingUserID int) ([]models.UserBasic, error) {
	query = normalizeInviteUserSearch(query)
	if len(query) < 2 {
		return []models.UserBasic{}, nil
	}
	users, err := s.users.List(query, 1, 10)
	if err != nil {
		return nil, err
	}
	out := make([]models.UserBasic, 0, len(users))
	for _, u := range users {
		if u.ID == requestingUserID {
			continue
		}
		out = append(out, models.UserBasic{ID: u.ID, Username: u.Username, Email: u.Email})
	}
	return out, nil
}

func normalizeInviteUserSearch(query string) string {
	query = strings.TrimSpace(query)
	if strings.HasPrefix(strings.ToLower(query), "ww@") {
		query = query[3:]
	}
	return strings.TrimSpace(query)
}

// SendInvitation creates a pending invitation from inviterID to inviteeID on the playlist.
// Returns (invitationID, httpStatus, error).
func (s *InvitationService) SendInvitation(playlistID, inviterID, inviteeID int) (int, int, error) {
	// 1. Playlist must exist
	pl, err := s.playlists.GetByID(playlistID)
	if err == sql.ErrNoRows || pl == nil {
		return 0, 404, fmt.Errorf("playlist not found")
	}
	if err != nil {
		return 0, 500, err
	}

	// 2. Inviter must be leader or co-lead
	if pl.UserID != inviterID {
		// Check if they are a co-lead
		team, err := s.teams.FindByPlaylistID(playlistID)
		if err != nil {
			return 0, 500, err
		}
		if team == nil || !utils.ContainsInt(utils.ParseIntSlice(team.CoLeadsRaw.String), inviterID) {
			return 0, 403, fmt.Errorf("only the leader or co-lead can invite members")
		}
	}

	// 3. Cannot invite yourself
	if inviterID == inviteeID {
		return 0, 400, fmt.Errorf("cannot invite yourself")
	}

	// 4. Invitee must exist
	invitee, err := s.users.FindByID(inviteeID)
	if err != nil || invitee == nil {
		return 0, 404, fmt.Errorf("user not found")
	}

	// 5. Invitee must not already be a playlist member
	member, err := s.teams.GetMember(playlistID, inviteeID)
	if err != nil {
		return 0, 500, err
	}
	if member != nil {
		return 0, 409, fmt.Errorf("user is already a member of this playlist")
	}

	// Also check playlist_teams.members JSON array (in case the user joined but has no role row)
	if pl.PlaylistTeamID.Valid {
		team, err := s.teams.GetByID(int(pl.PlaylistTeamID.Int64))
		if err != nil {
			return 0, 500, err
		}
		if team != nil && utils.ContainsInt(utils.ParseIntSlice(team.MembersRaw.String), inviteeID) {
			return 0, 409, fmt.Errorf("user is already a member of this playlist")
		}
	}
	// Also block inviting the playlist owner
	if pl.UserID == inviteeID {
		return 0, 409, fmt.Errorf("user is already a member of this playlist")
	}

	// 6. No duplicate pending invitation
	existing, err := s.invitations.FindPendingByPlaylistAndInvitee(playlistID, inviteeID)
	if err != nil {
		return 0, 500, err
	}
	if existing != nil {
		return 0, 409, fmt.Errorf("an invitation is already pending for this user")
	}

	// 7. Create invitation
	invitationID, err := s.invitations.CreateInvitation(playlistID, inviterID, inviteeID)
	if err != nil {
		return 0, 500, err
	}

	// 8. Notify invitee (fire and forget — failure is logged, not fatal)
	inviter, _ := s.users.FindByID(inviterID)
	inviterName := "Someone"
	if inviter != nil {
		inviterName = inviter.Username
	}
	s.notif.NotifyPlaylistInvite(invitationID, playlistID, pl.PlaylistName, inviterName, inviteeID)

	return invitationID, 201, nil
}

// AcceptInvitation accepts a pending invitation, adding the invitee to the playlist team.
func (s *InvitationService) AcceptInvitation(invitationID, inviteeID int) (int, error) {
	inv, err := s.invitations.FindPendingByID(invitationID)
	if err != nil {
		return 500, err
	}
	if inv == nil {
		return 404, fmt.Errorf("invitation not found or already actioned")
	}
	if inv.InviteeID != inviteeID {
		return 403, fmt.Errorf("this invitation was not sent to you")
	}

	// Load playlist to get owner + team ID
	pl, err := s.playlists.GetByID(inv.PlaylistID)
	if err != nil || pl == nil {
		return 500, fmt.Errorf("playlist not found")
	}

	// Join the team (mirrors PlaylistService.Join without the share-token lookup)
	if !pl.PlaylistTeamID.Valid {
		// Team does not exist yet — create it
		newTeamID, err := s.teams.Create(inv.PlaylistID, pl.UserID, []int{pl.UserID, inviteeID})
		if err != nil {
			return 500, err
		}
		if err := s.playlists.SetTeamID(inv.PlaylistID, newTeamID); err != nil {
			return 500, err
		}
	} else {
		team, err := s.teams.GetByID(int(pl.PlaylistTeamID.Int64))
		if err != nil || team == nil {
			return 500, fmt.Errorf("failed to load playlist team")
		}
		members := utils.ParseIntSlice(team.MembersRaw.String)
		if !utils.ContainsInt(members, inviteeID) {
			members = append(members, inviteeID)
			if err := s.teams.UpdateMembers(team.ID, members); err != nil {
				return 500, err
			}
		}
	}

	// Initialise playlist_members row with default role 'singer'
	_ = s.teams.UpsertMember(inv.PlaylistID, inviteeID, "singer")

	// Mark accepted
	if err := s.invitations.UpdateStatus(invitationID, "accepted"); err != nil {
		return 500, err
	}

	// Notify inviter
	invitee, _ := s.users.FindByID(inviteeID)
	inviteeName := "Someone"
	if invitee != nil {
		inviteeName = invitee.Username
	}
	s.notif.NotifyInvitationAccepted(inv.PlaylistID, pl.PlaylistName, inviteeName, inv.InviterID)

	return 200, nil
}

// DeclineInvitation declines a pending invitation.
func (s *InvitationService) DeclineInvitation(invitationID, inviteeID int) (int, error) {
	inv, err := s.invitations.FindPendingByID(invitationID)
	if err != nil {
		return 500, err
	}
	if inv == nil {
		return 404, fmt.Errorf("invitation not found or already actioned")
	}
	if inv.InviteeID != inviteeID {
		return 403, fmt.Errorf("this invitation was not sent to you")
	}
	if err := s.invitations.UpdateStatus(invitationID, "declined"); err != nil {
		return 500, err
	}
	return 200, nil
}
