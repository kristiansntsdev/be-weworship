package services

import (
	"database/sql"
	"strconv"
	"strings"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTeam builds a minimal PlaylistTeam for use in tests.
// members and coLeads are Go slices; the helper encodes them as JSON strings.
func makeTeam(id, playlistID, leadID int, members, coLeads []int) *models.PlaylistTeam {
	return &models.PlaylistTeam{
		ID:         id,
		PlaylistID: playlistID,
		LeadID:     leadID,
		MembersRaw: sql.NullString{String: intsToJSON(members), Valid: true},
		CoLeadsRaw: sql.NullString{String: intsToJSON(coLeads), Valid: true},
	}
}

// intsToJSON converts a slice of ints to a JSON array string like "[1,2,3]".
func intsToJSON(ids []int) string {
	if len(ids) == 0 {
		return "[]"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// ── RemoveMember ──────────────────────────────────────────────────────────────

func TestTeamService_RemoveMember_Lead(t *testing.T) {
	updateCalled := false
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7, 8}, nil), nil
		},
		UpdateMembersFn: func(_ int, members []int) error {
			updateCalled = true
			assert.NotContains(t, members, 7, "removed member must not be in updated list")
			return nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.RemoveMember(1, 7 /*memberID*/, 5 /*requesterID=lead*/)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, updateCalled)
}

func TestTeamService_RemoveMember_NotLead(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7, 8}, nil), nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.RemoveMember(1, 7, 8 /*requesterID≠lead*/)

	assert.Error(t, err)
	assert.Equal(t, 403, status)
}

func TestTeamService_RemoveMember_NotInTeam(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5, []int{5, 7}, nil), nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.RemoveMember(1, 99 /*not in members*/, 5 /*lead*/)

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

// ── Leave ─────────────────────────────────────────────────────────────────────

func TestTeamService_Leave_Member(t *testing.T) {
	updateCalled := false
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7, 8}, nil), nil
		},
		UpdateMembersFn: func(_ int, members []int) error {
			updateCalled = true
			assert.NotContains(t, members, 7)
			return nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.Leave(1, 7 /*requesterID=member, not lead*/)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, updateCalled)
}

func TestTeamService_Leave_LeadBlocked(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7}, nil), nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.Leave(1, 5 /*requesterID=lead — blocked*/)

	assert.Error(t, err)
	assert.Equal(t, 403, status)
}

// ── PromoteToCoLead ───────────────────────────────────────────────────────────

func TestTeamService_PromoteToCoLead_Success(t *testing.T) {
	updateCalled := false
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7, 8}, []int{} /*no co-leads*/), nil
		},
		UpdateCoLeadsFn: func(_ int, coLeads []int) error {
			updateCalled = true
			assert.Contains(t, coLeads, 7)
			return nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.PromoteToCoLead(1, 5 /*lead*/, 7 /*member to promote*/)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, updateCalled)
}

func TestTeamService_PromoteToCoLead_AlreadyCoLead(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5, []int{5, 7}, []int{7} /*7 is already co-lead*/), nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.PromoteToCoLead(1, 5, 7)

	assert.Error(t, err)
	assert.Equal(t, 400, status)
}

func TestTeamService_PromoteToCoLead_NotLead(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5, []int{5, 7, 8}, nil), nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.PromoteToCoLead(1, 8 /*not lead*/, 7)

	assert.Error(t, err)
	assert.Equal(t, 403, status)
}

// ── DemoteCoLead ──────────────────────────────────────────────────────────────

func TestTeamService_DemoteCoLead_Success(t *testing.T) {
	updateCalled := false
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5, []int{5, 7}, []int{7} /*7 is co-lead*/), nil
		},
		UpdateCoLeadsFn: func(_ int, coLeads []int) error {
			updateCalled = true
			assert.NotContains(t, coLeads, 7)
			return nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.DemoteCoLead(1, 5 /*lead*/, 7 /*co-lead to demote*/)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, updateCalled)
}

func TestTeamService_DemoteCoLead_NotCoLead(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetByIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5, []int{5, 7}, []int{} /*no co-leads*/), nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.DemoteCoLead(1, 5, 7 /*not a co-lead*/)

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

// ── SetMemberRole ─────────────────────────────────────────────────────────────

func TestTeamService_SetMemberRole_InvalidRole(t *testing.T) {
	svc := &TeamService{teams: &mocks.TeamRepo{}, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.SetMemberRole(10, 5, 7, "drummer")

	assert.Error(t, err)
	assert.Equal(t, 400, status)
}

func TestTeamService_SetMemberRole_Success_Lead(t *testing.T) {
	upsertCalled := false
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7}, nil), nil
		},
		GetMemberFn: func(playlistID, userID int) (*models.PlaylistMember, error) {
			// requester (5) and target (7) are both members with role singer
			return &models.PlaylistMember{PlaylistID: playlistID, UserID: userID, Role: "singer"}, nil
		},
		UpsertMemberFn: func(int, int, string) error {
			upsertCalled = true
			return nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	// Lead (5) promotes member (7) to musician
	status, err := svc.SetMemberRole(10, 5 /*requesterID=lead*/, 7, "musician")

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, upsertCalled)
}

func TestTeamService_SetMemberRole_NotAuthorised(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7, 8}, nil), nil
		},
		GetMemberFn: func(playlistID, userID int) (*models.PlaylistMember, error) {
			// requester (8) is a regular singer, not lead or MD
			return &models.PlaylistMember{PlaylistID: playlistID, UserID: userID, Role: "singer"}, nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.SetMemberRole(10, 8 /*requesterID≠lead, role=singer*/, 7, "musician")

	assert.Error(t, err)
	assert.Equal(t, 403, status)
}

func TestTeamService_SetMemberRole_MaxMD(t *testing.T) {
	callCount := 0
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 7}, nil), nil
		},
		GetMemberFn: func(playlistID, userID int) (*models.PlaylistMember, error) {
			callCount++
			// Both requester and target return singer so we can test the MD count
			// but requester is lead (5), so GetMember for requester still returns singer (isLead overrides)
			return &models.PlaylistMember{PlaylistID: playlistID, UserID: userID, Role: "singer"}, nil
		},
		CountByRoleFn: func(int, string) (int, error) { return 1, nil }, // already 1 MD
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	// Lead (5) tries to assign md to member (7), but another MD already exists
	status, err := svc.SetMemberRole(10, 5 /*lead*/, 7, "md")

	assert.Error(t, err)
	assert.Equal(t, 409, status)
}

func TestTeamService_SetMemberRole_TargetNotMember(t *testing.T) {
	callNum := 0
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5, []int{5, 7}, nil), nil
		},
		GetMemberFn: func(playlistID, userID int) (*models.PlaylistMember, error) {
			callNum++
			if callNum == 1 {
				// First call: requester (lead 5) — return any member to confirm auth
				return &models.PlaylistMember{UserID: userID, Role: "singer"}, nil
			}
			// Second call: target (99) — not in playlist_members
			return nil, nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	status, err := svc.SetMemberRole(10, 5 /*lead*/, 99 /*not a member*/, "musician")

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

func TestTeamService_SetMemberRole_MDByMD(t *testing.T) {
	// An existing MD (not the lead) can also set roles
	upsertCalled := false
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return makeTeam(1, 10, 5 /*lead*/, []int{5, 6, 7}, nil), nil
		},
		GetMemberFn: func(playlistID, userID int) (*models.PlaylistMember, error) {
			if userID == 6 { // requester is MD
				return &models.PlaylistMember{UserID: 6, Role: "md"}, nil
			}
			return &models.PlaylistMember{UserID: userID, Role: "singer"}, nil
		},
		UpsertMemberFn: func(int, int, string) error {
			upsertCalled = true
			return nil
		},
	}
	svc := &TeamService{teams: mockTeams, users: &mocks.AuthRepo{}, playlists: &mocks.PlaylistRepo{}}
	// MD (6) assigns musician role to member (7)
	status, err := svc.SetMemberRole(10, 6 /*MD*/, 7, "musician")

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.True(t, upsertCalled)
}
