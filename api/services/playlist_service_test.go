package services

import (
	"database/sql"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestPlaylistService_Create_Success(t *testing.T) {
	var storedPublic *bool
	mockPlaylists := &mocks.PlaylistRepo{
		NameExistsForUserFn: func(int, string) (bool, error) { return false, nil },
		CreateFn: func(_ int, _ string, _ []int, isPublic bool) (int, error) {
			storedPublic = &isPublic
			return 10, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	data, status, err := svc.Create(1, "Sunday Set", []int{1, 2}, false)

	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Equal(t, 10, data["id"])
	assert.Equal(t, "Sunday Set", data["playlist_name"])
	require.NotNil(t, storedPublic)
	assert.False(t, *storedPublic, "playlists are private unless asked otherwise")
	assert.Equal(t, false, data["is_public"])
}

func TestPlaylistService_Create_Public(t *testing.T) {
	var storedPublic bool
	mockPlaylists := &mocks.PlaylistRepo{
		NameExistsForUserFn: func(int, string) (bool, error) { return false, nil },
		CreateFn: func(_ int, _ string, _ []int, isPublic bool) (int, error) {
			storedPublic = isPublic
			return 11, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	data, status, err := svc.Create(1, "Open Set", nil, true)

	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.True(t, storedPublic)
	assert.Equal(t, true, data["is_public"])
}

func TestPlaylistService_Create_EmptyName(t *testing.T) {
	svc := &PlaylistService{playlists: &mocks.PlaylistRepo{}, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.Create(1, "   ", nil, false)

	assert.Error(t, err)
	assert.Equal(t, 400, status)
}

func TestPlaylistService_Create_DuplicateName(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		NameExistsForUserFn: func(int, string) (bool, error) { return true, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.Create(1, "Sunday Set", nil, false)

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

func TestPlaylistService_GetByIDWithAccess_PublicPlaylistViewer(t *testing.T) {
	const (
		playlistID = 5
		ownerID    = 10
		strangerID = 77
	)
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{
				ID: playlistID, UserID: ownerID, PlaylistName: "Sunday Set", IsPublic: true,
				// Shared with a team too: its token must not reach a stranger.
				IsShared:   true,
				ShareToken: sql.NullString{String: "5-secret", Valid: true},
			}, nil
		},
		GetSongKeysFn: func(int) (map[int]string, error) { return map[int]string{1: "G"}, nil },
		GetOwnerFn: func(id int) (*repositories.PlaylistOwner, error) {
			assert.Equal(t, ownerID, id)
			return &repositories.PlaylistOwner{Username: "rina", DisplayName: "Rina S", Verified: true}, nil
		},
	}
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return &models.PlaylistTeam{ID: 1, LeadID: ownerID, MembersRaw: sql.NullString{String: "[10,42]", Valid: true}}, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: mockTeams, songs: &mocks.SongRepo{}}
	data, status, err := svc.GetByIDWithAccess(playlistID, strangerID)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "viewer", data["access_type"])
	assert.Equal(t, map[string]string{"1": "G"}, data["song_keys"], "viewers play in the owner's keys")
	assert.Equal(t, "rina", data["owner"].(map[string]any)["username"])
	for _, k := range []string{"share_token", "sharable_link", "playlist_team_id"} {
		assert.NotContains(t, data, k)
	}
}

func TestPlaylistService_GetByIDWithAccess_MemberStillGetsShareToken(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{ID: 5, UserID: 10, IsPublic: true, IsShared: true, ShareToken: sql.NullString{String: "5-secret", Valid: true}}, nil
		},
		GetSongKeysFn: func(int) (map[int]string, error) { return nil, nil },
	}
	mockTeams := &mocks.TeamRepo{
		FindByPlaylistIDFn: func(int) (*models.PlaylistTeam, error) {
			return &models.PlaylistTeam{ID: 1, LeadID: 10, MembersRaw: sql.NullString{String: "[42]", Valid: true}}, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: mockTeams, songs: &mocks.SongRepo{}}
	data, _, err := svc.GetByIDWithAccess(5, 42)

	require.NoError(t, err)
	assert.Equal(t, "member", data["access_type"], "membership wins over public access")
	assert.Equal(t, "5-secret", data["share_token"])
}

func TestPlaylistService_GetByIDWithAccess_PrivatePlaylistStranger(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{ID: 5, UserID: 10, IsPublic: false}, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.GetByIDWithAccess(5, 77)

	assert.Error(t, err)
	assert.Equal(t, 404, status)
}

// ── Copy ──────────────────────────────────────────────────────────────────────

func TestPlaylistService_Copy_PublicPlaylistKeepsSongsAndKeys(t *testing.T) {
	var created struct {
		userID   int
		name     string
		songs    []int
		isPublic bool
	}
	var keysFrom, keysTo int
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{ID: 5, UserID: 10, PlaylistName: "Sunday Set", IsPublic: true, SongsRaw: sql.NullString{String: "[3,1,2]", Valid: true}}, nil
		},
		// "(copy)" is taken, "(copy) 2" is free.
		NameExistsForUserFn: func(_ int, name string) (bool, error) { return name == "Sunday Set (copy)", nil },
		CreateFn: func(userID int, name string, songs []int, isPublic bool) (int, error) {
			created.userID, created.name, created.songs, created.isPublic = userID, name, songs, isPublic
			return 99, nil
		},
		CopySongKeysFn: func(from, to int) error {
			keysFrom, keysTo = from, to
			return nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	data, status, err := svc.Copy(5, 77)

	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Equal(t, 99, data["id"])
	assert.Equal(t, "Sunday Set (copy) 2", data["playlist_name"])
	assert.Equal(t, 77, created.userID)
	assert.Equal(t, []int{3, 1, 2}, created.songs, "same songs in the same order")
	assert.False(t, created.isPublic, "copies are private")
	assert.Equal(t, 5, keysFrom)
	assert.Equal(t, 99, keysTo)
}

func TestPlaylistService_Copy_PrivatePlaylistOfSomeoneElse(t *testing.T) {
	mockPlaylists := &mocks.PlaylistRepo{
		GetByIDFn: func(int) (*models.Playlist, error) {
			return &models.Playlist{ID: 5, UserID: 10, PlaylistName: "Secret", IsPublic: false}, nil
		},
		CreateFn: func(int, string, []int, bool) (int, error) {
			t.Fatal("must not create a copy without read access")
			return 0, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	_, status, err := svc.Copy(5, 77)

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

// ── UpdateVisibility ──────────────────────────────────────────────────────────

func TestPlaylistService_UpdateVisibility_Owner(t *testing.T) {
	var gotID, gotUser int
	var gotPublic bool
	mockPlaylists := &mocks.PlaylistRepo{
		UpdateVisibilityFn: func(id, userID int, isPublic bool) (int64, error) {
			gotID, gotUser, gotPublic = id, userID, isPublic
			return 1, nil
		},
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	status, err := svc.UpdateVisibility(5, 42, true)

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, 5, gotID)
	assert.Equal(t, 42, gotUser)
	assert.True(t, gotPublic)
}

func TestPlaylistService_UpdateVisibility_NotOwnerOrMissing(t *testing.T) {
	// The repo scopes the UPDATE to the owner, so a non-owner (or a missing playlist) affects 0 rows.
	mockPlaylists := &mocks.PlaylistRepo{
		UpdateVisibilityFn: func(int, int, bool) (int64, error) { return 0, nil },
	}
	svc := &PlaylistService{playlists: mockPlaylists, teams: &mocks.TeamRepo{}, songs: &mocks.SongRepo{}}
	status, err := svc.UpdateVisibility(5, 99, true)

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
