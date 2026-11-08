package services

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileService_Search_MapsRowsWithoutEmail(t *testing.T) {
	var gotQuery string
	repo := &mocks.ProfileRepo{SearchFn: func(q string, limit int) ([]repositories.ProfileRow, error) {
		gotQuery = q
		assert.Equal(t, profileSearchLimit, limit)
		return []repositories.ProfileRow{
			{ID: 3, Username: "rina", DisplayName: "Rina S", AvatarURL: sql.NullString{String: "https://a/r.png", Valid: true}, Verified: true, PublicPlaylists: 2},
			{ID: 4, Username: "rino", DisplayName: "rino"},
		}, nil
	}}
	out, err := NewProfileService(repo).Search("  @ri  ")

	require.NoError(t, err)
	assert.Equal(t, "ri", gotQuery, "trims spaces and a leading @")
	require.Len(t, out, 2)
	assert.Equal(t, "Rina S", out[0].DisplayName)
	require.NotNil(t, out[0].AvatarURL)
	assert.Nil(t, out[1].AvatarURL)
	assert.Equal(t, 2, out[0].PublicPlaylists)

	buf, _ := json.Marshal(out)
	assert.NotContains(t, string(buf), "email")
	assert.NotContains(t, string(buf), `"id"`, "internal user ids are not exposed either")
}

func TestProfileService_Search_ShortQueryReturnsNothing(t *testing.T) {
	repo := &mocks.ProfileRepo{SearchFn: func(string, int) ([]repositories.ProfileRow, error) {
		t.Fatal("must not query the database for a 1-character search")
		return nil, nil
	}}
	out, err := NewProfileService(repo).Search("r")

	require.NoError(t, err)
	assert.NotNil(t, out, "empty list, not null")
	assert.Empty(t, out)
}

func TestProfileService_Get_ReturnsPublicPlaylistsWithSongCounts(t *testing.T) {
	updated := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	repo := &mocks.ProfileRepo{
		FindByUsernameFn: func(u string) (*repositories.ProfileRow, error) {
			assert.Equal(t, "Rina", u)
			return &repositories.ProfileRow{ID: 3, Username: "rina", DisplayName: "Rina S", Verified: true}, nil
		},
		PublicPlaylistsFn: func(userID int) ([]repositories.PublicPlaylistRow, error) {
			assert.Equal(t, 3, userID)
			return []repositories.PublicPlaylistRow{
				{ID: 10, PlaylistName: "Sunday Set", SongsRaw: sql.NullString{String: "[1,2,3]", Valid: true}, UpdatedAt: updated},
				{ID: 11, PlaylistName: "Empty"},
			}, nil
		},
	}
	p, err := NewProfileService(repo).Get("Rina")

	require.NoError(t, err)
	assert.Equal(t, "rina", p.Username)
	assert.True(t, p.Verified)
	require.Len(t, p.Playlists, 2)
	assert.Equal(t, 3, p.Playlists[0].SongCount)
	assert.Equal(t, 0, p.Playlists[1].SongCount)
	assert.Equal(t, updated, p.Playlists[0].UpdatedAt)
}

func TestProfileService_Get_UnknownOrInactiveUser(t *testing.T) {
	// FindByUsername only returns active users, so inactive and banned accounts look the same as missing ones.
	repo := &mocks.ProfileRepo{FindByUsernameFn: func(string) (*repositories.ProfileRow, error) { return nil, nil }}
	_, err := NewProfileService(repo).Get("ghost")
	assert.ErrorIs(t, err, ErrProfileNotFound)

	_, err = NewProfileService(repo).Get("  ")
	assert.ErrorIs(t, err, ErrProfileNotFound)
}
