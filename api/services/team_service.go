package services

import (
	"fmt"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type TeamService struct {
	teams     repositories.TeamRepoIface
	users     repositories.AuthRepoIface
	playlists repositories.PlaylistRepoIface
}

func NewTeamService(t *repositories.TeamRepository, u *repositories.AuthRepository, p *repositories.PlaylistRepository) *TeamService {
	return &TeamService{teams: t, users: u, playlists: p}
}

func (s *TeamService) ListByLead(userID int) ([]map[string]any, error) {
	rows, err := s.teams.ListByLeadID(userID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{"id": r.ID, "playlist_id": r.PlaylistID, "lead_id": r.LeadID, "members": utils.ParseIntSlice(r.MembersRaw.String), "createdAt": utils.NullableTime(r.CreatedAt), "updatedAt": utils.NullableTime(r.UpdatedAt)})
	}
	return out, nil
}

func (s *TeamService) GetByID(teamID int) (map[string]any, int, error) {
	team, err := s.teams.GetByID(teamID)
	if err != nil {
		return nil, 500, err
	}
	if team == nil {
		return nil, 404, fmt.Errorf("playlist team not found")
	}

	memberIDs := utils.ParseIntSlice(team.MembersRaw.String)
	coLeadIDs := utils.ParseIntSlice(team.CoLeadsRaw.String)

	// Fetch all users (leader + co-leads + members) in a single query
	seen := map[int]struct{}{}
	allIDs := []int{team.LeadID}
	seen[team.LeadID] = struct{}{}
	for _, id := range coLeadIDs {
		if _, ok := seen[id]; !ok {
			allIDs = append(allIDs, id)
			seen[id] = struct{}{}
		}
	}
	for _, id := range memberIDs {
		if _, ok := seen[id]; !ok {
			allIDs = append(allIDs, id)
			seen[id] = struct{}{}
		}
	}
	users, err := s.users.FindManyByIDs(allIDs)
	if err != nil {
		return nil, 500, err
	}
	userMap := make(map[int]map[string]any, len(users))
	for _, u := range users {
		userMap[u.ID] = map[string]any{"id": u.ID, "username": u.Username, "email": u.Email}
	}

	var leaderMap any
	if lm, ok := userMap[team.LeadID]; ok {
		leaderMap = lm
	}

	coLeadRows := make([]map[string]any, 0, len(coLeadIDs))
	for _, id := range coLeadIDs {
		if um, ok := userMap[id]; ok {
			coLeadRows = append(coLeadRows, um)
		}
	}

	memberRows := make([]map[string]any, 0, len(memberIDs))
	for _, id := range memberIDs {
		if um, ok := userMap[id]; ok {
			memberRows = append(memberRows, um)
		}
	}

	return map[string]any{
		"id":          team.ID,
		"playlist_id": team.PlaylistID,
		"lead_id":     team.LeadID,
		"leader":      leaderMap,
		"co_leads":    coLeadRows,
		"members":     memberRows,
		"createdAt":   utils.NullableTime(team.CreatedAt),
		"updatedAt":   utils.NullableTime(team.UpdatedAt),
	}, 200, nil
}

func (s *TeamService) RemoveMember(teamID, memberID, requesterID int) (int, error) {
	team, err := s.teams.GetByID(teamID)
	if err != nil || team == nil {
		return 404, fmt.Errorf("playlist team not found")
	}
	if team.LeadID != requesterID {
		return 403, fmt.Errorf("access denied. only team leader can remove members.")
	}
	members := utils.ParseIntSlice(team.MembersRaw.String)
	if !utils.ContainsInt(members, memberID) {
		return 404, fmt.Errorf("user is not a member of this team")
	}
	next := []int{}
	for _, id := range members {
		if id != memberID {
			next = append(next, id)
		}
	}
	if err := s.teams.UpdateMembers(teamID, next); err != nil {
		return 500, err
	}
	return 200, nil
}

func (s *TeamService) Leave(teamID, requesterID int) (int, error) {
	team, err := s.teams.GetByID(teamID)
	if err != nil || team == nil {
		return 404, fmt.Errorf("playlist team not found")
	}
	if team.LeadID == requesterID {
		return 403, fmt.Errorf("team leader cannot leave the team. transfer leadership or delete the team.")
	}
	members := utils.ParseIntSlice(team.MembersRaw.String)
	if !utils.ContainsInt(members, requesterID) {
		return 404, fmt.Errorf("you are not a member of this team")
	}
	next := []int{}
	for _, id := range members {
		if id != requesterID {
			next = append(next, id)
		}
	}
	if err := s.teams.UpdateMembers(teamID, next); err != nil {
		return 500, err
	}
	return 200, nil
}

func (s *TeamService) PromoteToCoLead(teamID, requesterID, memberID int) (int, error) {
	team, err := s.teams.GetByID(teamID)
	if err != nil || team == nil {
		return 404, fmt.Errorf("playlist team not found")
	}
	if team.LeadID != requesterID {
		return 403, fmt.Errorf("access denied. only team leader can promote co-leads.")
	}
	members := utils.ParseIntSlice(team.MembersRaw.String)
	if !utils.ContainsInt(members, memberID) {
		return 400, fmt.Errorf("user is not a member of this team")
	}
	coLeads := utils.ParseIntSlice(team.CoLeadsRaw.String)
	if utils.ContainsInt(coLeads, memberID) {
		return 400, fmt.Errorf("user is already a co-lead")
	}
	coLeads = append(coLeads, memberID)
	if err := s.teams.UpdateCoLeads(teamID, coLeads); err != nil {
		return 500, err
	}
	return 200, nil
}

func (s *TeamService) DemoteCoLead(teamID, requesterID, coLeadID int) (int, error) {
	team, err := s.teams.GetByID(teamID)
	if err != nil || team == nil {
		return 404, fmt.Errorf("playlist team not found")
	}
	if team.LeadID != requesterID {
		return 403, fmt.Errorf("access denied. only team leader can demote co-leads.")
	}
	coLeads := utils.ParseIntSlice(team.CoLeadsRaw.String)
	if !utils.ContainsInt(coLeads, coLeadID) {
		return 404, fmt.Errorf("user is not a co-lead of this team")
	}
	next := []int{}
	for _, id := range coLeads {
		if id != coLeadID {
			next = append(next, id)
		}
	}
	if err := s.teams.UpdateCoLeads(teamID, next); err != nil {
		return 500, err
	}
	return 200, nil
}

// ── Playlist member roles ─────────────────────────────────────────────────────

// ListMembers returns all playlist_members rows for a playlist, enriched with user info.
// Any authenticated team member may call this.
func (s *TeamService) ListMembers(playlistID int) ([]map[string]any, int, error) {
	rows, err := s.teams.ListMembers(playlistID)
	if err != nil {
		return nil, 500, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"user_id": r.UserID,
			"username": r.Username,
			"email":   r.Email,
			"role":    r.Role,
		})
	}
	return out, 200, nil
}

// SetMemberRole sets (or updates) the role for a member in a playlist.
// Only the team lead or an existing MD may assign roles.
// Business rules:
//   - role must be one of: singer, musician, md
//   - max 1 MD per playlist; promote request is rejected if another MD already exists
//   - target user must already be a member in playlist_members (via the backfill or join flow)
func (s *TeamService) SetMemberRole(playlistID, requesterID, targetUserID int, role string) (int, error) {
	// Validate role value
	switch role {
	case "singer", "musician", "md":
	default:
		return 400, fmt.Errorf("invalid role: must be 'singer', 'musician', or 'md'")
	}

	// Verify the team exists for this playlist
	team, err := s.teams.FindByPlaylistID(playlistID)
	if err != nil {
		return 500, err
	}
	if team == nil {
		return 404, fmt.Errorf("playlist team not found")
	}

	// Determine if requester is authorised (lead OR current MD)
	requesterMember, err := s.teams.GetMember(playlistID, requesterID)
	if err != nil {
		return 500, err
	}
	isLead := team.LeadID == requesterID
	isMD := requesterMember != nil && requesterMember.Role == "md"
	if !isLead && !isMD {
		return 403, fmt.Errorf("access denied: only the team lead or an MD can set member roles")
	}

	// Verify target user is a member of this playlist
	targetMember, err := s.teams.GetMember(playlistID, targetUserID)
	if err != nil {
		return 500, err
	}
	if targetMember == nil {
		return 404, fmt.Errorf("target user is not a member of this playlist")
	}

	// Enforce max-1-MD rule
	if role == "md" && (targetMember.Role != "md") {
		count, err := s.teams.CountByRole(playlistID, "md")
		if err != nil {
			return 500, err
		}
		if count >= 1 {
			return 409, fmt.Errorf("playlist already has an MD — demote the current MD first")
		}
	}

	if err := s.teams.UpsertMember(playlistID, targetUserID, role); err != nil {
		return 500, err
	}
	return 200, nil
}

func (s *TeamService) Delete(teamID, requesterID int) (int, error) {
	team, err := s.teams.GetByID(teamID)
	if err != nil || team == nil {
		return 404, fmt.Errorf("playlist team not found")
	}
	if team.LeadID != requesterID {
		return 403, fmt.Errorf("access denied. only team leader can delete the team.")
	}
	if err := s.playlists.ClearShareAndTeam(team.PlaylistID); err != nil {
		return 500, err
	}
	if err := s.teams.Delete(teamID); err != nil {
		return 500, err
	}
	return 200, nil
}
