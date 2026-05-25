package services

import (
	"database/sql"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestPlaylistService_Create_Success(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		NameExistsForUserFn: func(int, string) (bool, error) { return false, nil },
		CreateFn:            func(int, string, []int) (int, error) { return 10, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	data, status, err := svc.Create(1, "Sunday Set", []int{1, 2})

	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Equal(t, 10, data["id"])
	assert.Equal(t, "Sunday Set", data["playlist_name"])
}

func TestPlaylistService_Create_EmptyName(t *testing.T) {
	svc := &PlaylistService{playlists: &mocks.PlaylistRepo{}, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.Create(1, "   ", nil)

	assert.Error(t, err)
	assert.Equal(t, 400, status)
}

func TestPlaylistService_Create_DuplicateName(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		NameExistsForUserFn: func(int, string) (bool, error) { return true, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.Create(1, "Sunday Set", nil)

	assert.Error(t, err)
	assert.Equal(t, 409, status)
}

// ── GetByIDWithAccess ─────────────────────────────────────────────────────────

func TestPlaylistService_GetByIDWithAccess_Owner(t *testing.T) {
	const (
		playlistID = 5
		userID     = 42
	)
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{ID: playlistID, UserID: userID, PlaylistName: "Sunday Set"}, nil
		},
		GetSongKeysFn: func(int) (map[int]string, error) { return nil, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	data, status, err := svc.GetByIDWithAccess(playlistID, userID)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "owner", data["access_type"])
	assert.Equal(t, "Sunday Set", data["playlist_name"])
}

func TestPlaylistService_GetByIDWithAccess_Member(t *testing.T) {
	const (
		playlistID = 5
		ownerID    = 10
		memberID   = 42
	)
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{ID: playlistID, UserID: ownerID, PlaylistName: "Sunday Set"}, nil
		},
		GetSongKeysFn: func(int) (map[int]string, error) { return nil, nil },
	}
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return &models.PlaylistTeam{
				ID:         1,
				PlaylistID: playlistID,
				LeadID:     100,
				MembersRaw: sql.NullString{String: "[42]", Valid: true},
				CoLeadsRaw: sql.NullString{Valid: false},
			}, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: mockTeams, songs: &mocks.SongRepo{}}
	data, status, err := svc.GetByIDWithAccess(playlistID, memberID)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "member", data["access_type"])
}

func TestPlaylistService_GetByIDWithAccess_NotFound(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) { return nil, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.GetByIDWithAccess(99, 1)

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestPlaylistService_Delete_Found(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		DeleteFn: func(int, int) (int64, error) { return 1, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	status, err := svc.Delete(1, 1)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
}

func TestPlaylistService_Delete_NotFound(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		DeleteFn: func(int, int) (int64, error) { return 0, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	status, err := svc.Delete(99, 1)

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

// ── Join ──────────────────────────────────────────────────────────────────────

func TestPlaylistService_Join_InvalidToken(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		FindByShareTokenFn: func(string) (int, int, sql.NullInt64, error) {
			return 0, 0, sql.NullInt64{}, sql.ErrNoRows
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.Join("bad-token", 5)

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

func TestPlaylistService_Join_OwnerJoinsOwnPlaylist(t *testing.T) {
	// When the owner uses their own share link, we just record the event and return 201
	recordCalled := false
	mockPlaylists := &mocks.PlaylistRepo{
		FindByShareTokenFn: func(string) (int, int, sql.NullInt64, error) {
			return 7, 42, sql.NullInt64{Valid: false}, nil // playlistID=7, ownerID=42
		},
		RecordShareEventFn: func(int) { recordCalled = true },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	data, status, err := svc.Join("valid-token", 42 /* owner */)

	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Equal(t, 7, data["playlist_id"])
	assert.True(t, recordCalled)
}

// ── GetViewerRole ─────────────────────────────────────────────────────────────

func TestPlaylistService_GetViewerRole_Member(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetMemberFn: func(playlistID, userID int) (*models.PlaylistMember, error) {
			return &models.PlaylistMember{UserID: userID, PlaylistID: playlistID, Role: "singer"}, nil
		},
	}
	svc := &PlaylistService{playlists: &mocks.PlaylistRepo{}, teams: mockTeams, songs: &mocks.SongRepo{}}
	role := svc.GetViewerRole(5, 10)
	assert.Equal(t, "singer", role)
}

func TestPlaylistService_GetViewerRole_Guest_NoMembership(t *testing.T) {
	mockTeams := &mocks.TeamRepo{
		GetMemberFn: func(int, int) (*models.PlaylistMember, error) { return nil, nil },
	}
	svc := &PlaylistService{playlists: &mocks.PlaylistRepo{}, teams: mockTeams, songs: &mocks.SongRepo{}}
	role := svc.GetViewerRole(5, 99)
	assert.Equal(t, "guest", role)
}

func TestPlaylistService_GetViewerRole_Unauthenticated(t *testing.T) {
	getMemberCalled := false
	mockTeams := &mocks.TeamRepo{
		GetMemberFn: func(int, int) (*models.PlaylistMember, error) {
			getMemberCalled = true
			return nil, nil
		},
	}
	svc := &PlaylistService{playlists: &mocks.PlaylistRepo{}, teams: mockTeams, songs: &mocks.SongRepo{}}
	role := svc.GetViewerRole(5, 0) // userID=0 means unauthenticated

	assert.Equal(t, "guest", role)
	assert.False(t, getMemberCalled, "GetMember must NOT be called for unauthenticated requests")
}
