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

func TestAnalyticsService_RecordSession_ReturnsRepoError(t *testing.T) {
	expectedErr := fmt.Errorf("insert failed")
	mockRepo := &mocks.AnalyticsRepo{
		RecordSessionFn: func(*int, string, string, string) error {
			return expectedErr
		},
	}
	svc := &AnalyticsService{repo: mockRepo}

	err := svc.RecordSession(nil, "ios", "1.0.0", "iOS 18")

	assert.ErrorIs(t, err, expectedErr)
}

func TestAnalyticsService_RecordPerformance_ForwardsPayload(t *testing.T) {
	userID := 7
	endpoint := "/api/songs/export"
	screenName := "library"
	durationMs := 120
	statusCode := 200
	called := false
	mockRepo := &mocks.AnalyticsRepo{
		RecordPerformanceFn: func(receivedUserID *int, platform, metricType string, receivedEndpoint, receivedScreenName *string, receivedDurationMs, receivedStatusCode *int, appVersion, deviceOS string) error {
			called = true
			require.NotNil(t, receivedUserID)
			assert.Equal(t, userID, *receivedUserID)
			assert.Equal(t, "android", platform)
			assert.Equal(t, "api_response", metricType)
			require.NotNil(t, receivedEndpoint)
			assert.Equal(t, endpoint, *receivedEndpoint)
			require.NotNil(t, receivedScreenName)
			assert.Equal(t, screenName, *receivedScreenName)
			require.NotNil(t, receivedDurationMs)
			assert.Equal(t, durationMs, *receivedDurationMs)
			require.NotNil(t, receivedStatusCode)
			assert.Equal(t, statusCode, *receivedStatusCode)
			assert.Equal(t, "1.0.0", appVersion)
			assert.Equal(t, "Android 15", deviceOS)
			return nil
		},
	}
	svc := &AnalyticsService{repo: mockRepo}

	err := svc.RecordPerformance(&userID, "android", "api_response", &endpoint, &screenName, &durationMs, &statusCode, "1.0.0", "Android 15")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestAnalyticsService_RecordPerformance_ReturnsRepoError(t *testing.T) {
	expectedErr := fmt.Errorf("insert failed")
	durationMs := 100
	mockRepo := &mocks.AnalyticsRepo{
		RecordPerformanceFn: func(*int, string, string, *string, *string, *int, *int, string, string) error {
			return expectedErr
		},
	}
	svc := &AnalyticsService{repo: mockRepo}

	err := svc.RecordPerformance(nil, "ios", "screen_load", nil, nil, &durationMs, nil, "1.0.0", "iOS 18")

	assert.ErrorIs(t, err, expectedErr)
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
