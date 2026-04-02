package services

import (
	"fmt"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type TeamService struct {
	teams     *repositories.TeamRepository
	users     *repositories.AuthRepository
	playlists *repositories.PlaylistRepository
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
		userMap[u.ID] = map[string]any{"id": u.ID, "name": u.Name, "email": u.Email}
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
