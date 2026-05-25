package services

import (
	"fmt"
	"testing"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── TopSongs ──────────────────────────────────────────────────────────────────

func TestAnalyticsService_TopSongs_Success(t *testing.T) {
	id1, id2 := 1, 2
	mockRepo := &mocks.AnalyticsRepo{
		TopSongsFn: func(days, limit int) ([]repositories.TopSongRow, error) {
			return []repositories.TopSongRow{
				{SongID: &id1, EventType: "play", Count: 50},
				{SongID: &id2, EventType: "play", Count: 30},
			}, nil
		},
	}
	svc := &AnalyticsService{repo: mockRepo}
	result, err := svc.TopSongs(30, 10)

	require.NoError(t, err)
	rows, ok := result.([]repositories.TopSongRow)
	require.True(t, ok)
	assert.Len(t, rows, 2)
	assert.Equal(t, 50, rows[0].Count)
}

func TestAnalyticsService_TopSongs_RepoError(t *testing.T) {
	mockRepo := &mocks.AnalyticsRepo{
		TopSongsFn: func(int, int) ([]repositories.TopSongRow, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}
	svc := &AnalyticsService{repo: mockRepo}
	_, err := svc.TopSongs(30, 10)
	assert.Error(t, err)
}

// ── UserStats ─────────────────────────────────────────────────────────────────

func TestAnalyticsService_UserStats_Success(t *testing.T) {
	mockRepo := &mocks.AnalyticsRepo{
		TotalUsersFn: func() (int, error) { return 100, nil },
		DAUTodayFn:   func() (int, error) { return 20, nil },
		MAUFn:        func() (int, error) { return 60, nil },
	}
	svc := &AnalyticsService{repo: mockRepo}
	stats, err := svc.UserStats(30)

	require.NoError(t, err)
	assert.Equal(t, 100, stats["total_users"])
	assert.Equal(t, 20, stats["dau"])
	assert.Equal(t, 60, stats["mau"])
}

func TestAnalyticsService_UserStats_TotalUsersError(t *testing.T) {
	mockRepo := &mocks.AnalyticsRepo{
		TotalUsersFn: func() (int, error) { return 0, fmt.Errorf("connection refused") },
	}
	svc := &AnalyticsService{repo: mockRepo}
	_, err := svc.UserStats(30)
	assert.Error(t, err)
}

func TestAnalyticsService_UserStats_DAUError(t *testing.T) {
	mockRepo := &mocks.AnalyticsRepo{
		TotalUsersFn: func() (int, error) { return 100, nil },
		DAUTodayFn:   func() (int, error) { return 0, fmt.Errorf("query failed") },
	}
	svc := &AnalyticsService{repo: mockRepo}
	_, err := svc.UserStats(30)
	assert.Error(t, err)
}

// ── RecordSearch ──────────────────────────────────────────────────────────────

func TestAnalyticsService_RecordSearch_CallsRepo(t *testing.T) {
	called := false
	mockRepo := &mocks.AnalyticsRepo{
		RecordSearchLogFn: func(*int, string, *string, int, string) {
			called = true
		},
	}
	svc := &AnalyticsService{repo: mockRepo}
	svc.RecordSearch(nil, "amazing grace", nil, 5, "mobile")
	assert.True(t, called, "RecordSearchLog must be called")
}

func TestAnalyticsService_RecordSearch_FiltersSerialised(t *testing.T) {
	var receivedJSON *string
	mockRepo := &mocks.AnalyticsRepo{
		RecordSearchLogFn: func(_ *int, _ string, filtersJSON *string, _ int, _ string) {
			receivedJSON = filtersJSON
		},
	}
	svc := &AnalyticsService{repo: mockRepo}
	svc.RecordSearch(nil, "query", map[string]any{"base_chord": "C"}, 3, "web")

	require.NotNil(t, receivedJSON)
	assert.Contains(t, *receivedJSON, "base_chord")
}

func TestAnalyticsService_RecordSearch_NilFilters(t *testing.T) {
	var receivedJSON *string
	mockRepo := &mocks.AnalyticsRepo{
		RecordSearchLogFn: func(_ *int, _ string, filtersJSON *string, _ int, _ string) {
			receivedJSON = filtersJSON
		},
	}
	svc := &AnalyticsService{repo: mockRepo}
	svc.RecordSearch(nil, "query", nil, 0, "web")
	assert.Nil(t, receivedJSON, "nil filters map must produce nil JSON")
}

// ── TopSearches ───────────────────────────────────────────────────────────────

func TestAnalyticsService_TopSearches_Success(t *testing.T) {
	mockRepo := &mocks.AnalyticsRepo{
		TopSearchesFn: func(int, int) ([]repositories.TopSearchRow, error) {
			return []repositories.TopSearchRow{{Query: "grace", Count: 10}}, nil
		},
	}
	svc := &AnalyticsService{repo: mockRepo}
	result, err := svc.TopSearches(7, 5)
	require.NoError(t, err)
	rows := result.([]repositories.TopSearchRow)
	assert.Len(t, rows, 1)
}
