package services

import (
"encoding/json"

"be-songbanks-v1/api/repositories"
)

type AnalyticsService struct {
repo *repositories.AnalyticsRepository
}

func NewAnalyticsService(repo *repositories.AnalyticsRepository) *AnalyticsService {
return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) RecordSongEvent(songID, userID *int, eventType, platform string, durationMs *int) {
go s.repo.RecordSongEvent(songID, userID, eventType, platform, durationMs)
}

func (s *AnalyticsService) RecordSearch(userID *int, query string, filters map[string]any, resultsCount int, platform string) {
var filtersJSON *string
if len(filters) > 0 {
if b, err := json.Marshal(filters); err == nil {
s := string(b)
filtersJSON = &s
}
}
go s.repo.RecordSearchLog(userID, query, filtersJSON, resultsCount, platform)
}

func (s *AnalyticsService) RecordSession(userID *int, platform, appVersion, deviceOS string) {
go s.repo.RecordSession(userID, platform, appVersion, deviceOS)
}

func (s *AnalyticsService) RecordPerformance(userID *int, platform, metricType string, endpoint, screenName *string, durationMs, statusCode *int, appVersion, deviceOS string) {
go s.repo.RecordPerformance(userID, platform, metricType, endpoint, screenName, durationMs, statusCode, appVersion, deviceOS)
}

// Admin analytics aggregates

func (s *AnalyticsService) TopSongs(days, limit int) (any, error) {
return s.repo.TopSongs(days, limit)
}

func (s *AnalyticsService) UserStats(days int) (map[string]any, error) {
newUsers, err := s.repo.NewUsersPerDay(days)
if err != nil {
return nil, err
}
dau, err := s.repo.DAU(days)
if err != nil {
return nil, err
}
return map[string]any{"new_users": newUsers, "dau": dau}, nil
}

func (s *AnalyticsService) TopSearches(days, limit int) (any, error) {
return s.repo.TopSearches(days, limit)
}

func (s *AnalyticsService) SessionsByPlatform(days int) (any, error) {
return s.repo.SessionsByPlatform(days)
}

func (s *AnalyticsService) PerformanceSummary(days int) (any, error) {
return s.repo.PerformanceSummary(days)
}
