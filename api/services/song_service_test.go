package services

import (
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestSongService_GetByID_Found(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		GetByIDFn: func(id int) (*models.Song, error) {
			return &models.Song{ID: id, Title: "Amazing Grace"}, nil
		},
	}
	mockTags := &mocks.TagRepo{
		GetTagsForSongFn: func(int) ([]models.Tag, error) { return nil, nil },
	}
	svc := &SongService{songs: mockSongs, tags: mockTags}
	data, found, err := svc.GetByID(1)

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Amazing Grace", data["title"])
}

func TestSongService_GetByID_NotFound(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		GetByIDFn: func(int) (*models.Song, error) { return nil, nil },
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	_, found, err := svc.GetByID(99)

	require.NoError(t, err)
	assert.False(t, found)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestSongService_Create_NewTag(t *testing.T) {
	createTagCalled := false
	assignCalled := false
	mockSongs := &mocks.SongRepo{
		CreateFn: func(string, string, *string, *int, *string, *string, bool, *string, string) (int, error) {
			return 42, nil
		},
		AssignSongTagFn: func(songID, tagID int) error {
			assignCalled = true
			return nil
		},
	}
	mockTags := &mocks.TagRepo{
		FindByNameFn: func(string) (*models.Tag, error) { return nil, nil },
		CreateFn: func(string) (int, error) {
			createTagCalled = true
			return 5, nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: mockTags}
	result, err := svc.Create("Amazing Grace", "Chris Tomlin", nil, nil, nil, nil, false, nil, []string{"worship"})

	require.NoError(t, err)
	assert.Equal(t, 42, result["id"])
	assert.True(t, createTagCalled, "tag must be created when not found")
	assert.True(t, assignCalled, "tag must be assigned to song")
}

func TestSongService_Create_ExistingTag(t *testing.T) {
	createTagCalled := false
	mockSongs := &mocks.SongRepo{
		CreateFn: func(string, string, *string, *int, *string, *string, bool, *string, string) (int, error) {
			return 10, nil
		},
		AssignSongTagFn: func(int, int) error { return nil },
	}
	mockTags := &mocks.TagRepo{
		FindByNameFn: func(string) (*models.Tag, error) {
			return &models.Tag{ID: 3, Name: "worship"}, nil
		},
		CreateFn: func(string) (int, error) {
			createTagCalled = true
			return 0, nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: mockTags}
	_, err := svc.Create("Holy Holy Holy", nil, nil, nil, nil, nil, false, nil, []string{"worship"})

	require.NoError(t, err)
	assert.False(t, createTagCalled, "tag.Create must NOT be called when tag already exists")
}

func TestSongService_Create_NoTags(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		CreateFn: func(string, string, *string, *int, *string, *string, bool, *string, string) (int, error) {
			return 7, nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	result, err := svc.Create("No Tag Song", nil, nil, nil, nil, nil, false, nil, []string{})

	require.NoError(t, err)
	assert.Equal(t, 7, result["id"])
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSongService_Delete_Found(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		DeleteByIDFn: func(int) (int64, error) { return 1, nil },
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	deleted, err := svc.Delete(1)

	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestSongService_Delete_NotFound(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		DeleteByIDFn: func(int) (int64, error) { return 0, nil },
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	deleted, err := svc.Delete(99)

	require.NoError(t, err)
	assert.False(t, deleted)
}

// ── RequestSong ───────────────────────────────────────────────────────────────

func TestSongService_RequestSong_Success(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		CreateSongRequestFn: func(userID int, title, refLink, lyricsType, lyrics string) (*models.SongRequest, error) {
			return &models.SongRequest{ID: 1, UserID: userID, SongTitle: title, ReferenceLink: refLink, LyricsType: lyricsType, Lyrics: lyrics}, nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	req, err := svc.RequestSong(5, "Amazing Grace", "https://youtube.com/xyz", "lyrics_chords", "[G]Amazing grace")

	require.NoError(t, err)
	assert.Equal(t, "Amazing Grace", req.SongTitle)
	assert.Equal(t, 5, req.UserID)
	assert.Equal(t, "lyrics_chords", req.LyricsType)
	assert.Equal(t, "[G]Amazing grace", req.Lyrics)
}

func TestSongService_RequestSong_EmptyTitle(t *testing.T) {
	svc := &SongService{songs: &mocks.SongRepo{}, tags: &mocks.TagRepo{}}
	_, err := svc.RequestSong(1, "   ", "https://youtube.com/xyz", "lyrics", "Some lyrics")
	assert.Error(t, err)
}

func TestSongService_RequestSong_EmptyReferenceLinkWithLyrics(t *testing.T) {
	mockSongs := &mocks.SongRepo{
		CreateSongRequestFn: func(userID int, title, refLink, lyricsType, lyrics string) (*models.SongRequest, error) {
			return &models.SongRequest{ID: 1, UserID: userID, SongTitle: title, ReferenceLink: refLink, LyricsType: lyricsType, Lyrics: lyrics}, nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	req, err := svc.RequestSong(1, "Some Song", "", "lyrics", "Plain lyrics")

	require.NoError(t, err)
	assert.Empty(t, req.ReferenceLink)
	assert.Equal(t, "lyrics", req.LyricsType)
}

func TestSongService_RequestSong_EmptyReferenceLinkAndLyrics(t *testing.T) {
	svc := &SongService{songs: &mocks.SongRepo{}, tags: &mocks.TagRepo{}}
	_, err := svc.RequestSong(1, "Some Song", "", "lyrics", "")
	assert.Error(t, err)
}

func TestSongService_RequestSong_InvalidLyricsType(t *testing.T) {
	svc := &SongService{songs: &mocks.SongRepo{}, tags: &mocks.TagRepo{}}
	_, err := svc.RequestSong(1, "Some Song", "", "tabs", "Some lyrics")
	assert.Error(t, err)
}

// ── UpdateSongRequestStatus ───────────────────────────────────────────────────

func TestSongService_UpdateSongRequestStatus_InvalidStatus(t *testing.T) {
	svc := &SongService{songs: &mocks.SongRepo{}, tags: &mocks.TagRepo{}}
	err := svc.UpdateSongRequestStatus(1, "unknown", "")
	assert.Error(t, err)
}

func TestSongService_UpdateSongRequestStatus_ValidStatus(t *testing.T) {
	var gotStatus string
	mockSongs := &mocks.SongRepo{
		UpdateSongRequestStatusFn: func(_ int, status, _ string) error {
			gotStatus = status
			return nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
	err := svc.UpdateSongRequestStatus(1, "approved", "Looks good")

	require.NoError(t, err)
	assert.Equal(t, "approved", gotStatus)
}

func TestSongService_UpdateSongRequestStatus_ApprovedClearsLyrics(t *testing.T) {
	clearCalled := false
	mockSongs := &mocks.SongRepo{
		UpdateSongRequestStatusFn: func(int, string, string) error { return nil },
		ClearSongRequestLyricsFn: func(id int) error {
			clearCalled = true
			assert.Equal(t, 1, id)
			return nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}

	err := svc.UpdateSongRequestStatus(1, "approved", "Song added")

	require.NoError(t, err)
	assert.True(t, clearCalled, "approved requests must drop stored lyrics content")
}

func TestSongService_UpdateSongRequestStatus_InProgressKeepsLyrics(t *testing.T) {
	clearCalled := false
	mockSongs := &mocks.SongRepo{
		UpdateSongRequestStatusFn: func(int, string, string) error { return nil },
		ClearSongRequestLyricsFn: func(int) error {
			clearCalled = true
			return nil
		},
	}
	svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}

	err := svc.UpdateSongRequestStatus(1, "in_progress", "")

	require.NoError(t, err)
	assert.False(t, clearCalled, "lyrics should remain while a request is still being worked")
}

func TestSongService_UpdateSongRequestStatus_AllValidStatuses(t *testing.T) {
	for _, status := range []string{"pending", "in_progress", "approved", "rejected"} {
		mockSongs := &mocks.SongRepo{
			UpdateSongRequestStatusFn: func(int, string, string) error { return nil },
		}
		svc := &SongService{songs: mockSongs, tags: &mocks.TagRepo{}}
		err := svc.UpdateSongRequestStatus(1, status, "")
		assert.NoError(t, err, "status %q must be accepted", status)
	}
}
