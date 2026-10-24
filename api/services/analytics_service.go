package services

import (
	"encoding/json"
	"strings"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type AnalyticsService struct {
	repo repositories.AnalyticsRepoIface
}

func NewAnalyticsService(repo *repositories.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) RecordSongEvent(songID, userID *int, eventType, platform string, durationMs *int) {
	s.repo.RecordSongEvent(songID, userID, eventType, platform, durationMs)
}

// RankedSongs returns top songs for home (popular views / most added). days=0 = all time.
func (s *AnalyticsService) RankedSongs(eventType string, days, limit int) ([]map[string]any, error) {
	rows, err := s.repo.RankedSongs(eventType, days, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		artist := strings.Join(utils.ParseArtists(r.Artist.String), ", ")
		var artistVal any
		if artist != "" {
			artistVal = artist
		}
		var keyVal any
		if r.Key.Valid && strings.TrimSpace(r.Key.String) != "" {
			keyVal = r.Key.String
		}
		out = append(out, map[string]any{
			"id":     r.ID,
			"title":  r.Title,
			"artist": artistVal,
			"key":    keyVal,
			"count":  r.Count,
		})
	}
	return out, nil
}

func (s *AnalyticsService) RecordSearch(userID *int, query string, filters map[string]any, resultsCount int, platform string) {
	var filtersJSON *string
	if len(filters) > 0 {
		if b, err := json.Marshal(filters); err == nil {
			s := string(b)
			filtersJSON = &s
		}
	}
	s.repo.RecordSearchLog(userID, query, filtersJSON, resultsCount, platform)
}

func (s *AnalyticsService) RecordSession(userID *int, platform, appVersion, deviceOS string) error {
	return s.repo.RecordSession(userID, platform, appVersion, deviceOS)
}

func (s *AnalyticsService) RecordPerformance(userID *int, platform, metricType string, endpoint, screenName *string, durationMs, statusCode *int, appVersion, deviceOS string) error {
	return s.repo.RecordPerformance(userID, platform, metricType, endpoint, screenName, durationMs, statusCode, appVersion, deviceOS)
}

// Admin analytics aggregates

func (s *AnalyticsService) TopSongs(days, limit int) (any, error) {
	rows, err := s.repo.TopSongs(days, limit)
	if err != nil {
		return nil, err
	}
	// Shape matches fe-admin dashboard DataTable / TopSongsChart (song_id, title, artist, play_count).
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		artist := strings.Join(utils.ParseArtists(r.Artist.String), ", ")
		var artistVal any
		if artist != "" {
			artistVal = artist
		}
		out = append(out, map[string]any{
			"song_id":    r.SongID,
			"title":      r.Title,
			"artist":     artistVal,
			"play_count": r.PlayCount,
		})
	}
	return out, nil
}

func (s *AnalyticsService) UserStats(days int) (map[string]any, error) {
	totalUsers, err := s.repo.TotalUsers()
	if err != nil {
		return nil, err
	}
	dau, err := s.repo.DAUToday()
	if err != nil {
		return nil, err
	}
	mau, err := s.repo.MAU()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total_users": totalUsers,
		"dau":         dau,
		"mau":         mau,
	}, nil
}

func (s *AnalyticsService) TopSearches(days, limit int) (any, error) {
	return s.repo.TopSearches(days, limit)
}

func (s *AnalyticsService) RecentSearches(userID, limit int) ([]string, error) {
	return s.repo.RecentSearchesByUser(userID, limit)
}

func (s *AnalyticsService) RecentViewedSongIDs(userID, limit int) ([]int, error) {
	return s.repo.RecentViewedSongIDs(userID, limit)
}

func (s *AnalyticsService) SessionsByPlatform(days int) (any, error) {
	return s.repo.SessionsByPlatform(days)
}

func (s *AnalyticsService) PerformanceSummary(days int) (any, error) {
	return s.repo.PerformanceSummary(days)
}
