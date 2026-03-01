package repositories

import (
"github.com/jmoiron/sqlx"
)

type AnalyticsRepository struct {
db *sqlx.DB
}

func NewAnalyticsRepository(db *sqlx.DB) *AnalyticsRepository {
return &AnalyticsRepository{db: db}
}

// ── Write Methods (fire-and-forget, errors are logged not returned) ───────────

func (r *AnalyticsRepository) RecordSongEvent(songID, userID *int, eventType, platform string, durationMs *int) {
r.db.Exec(r.db.Rebind(`INSERT INTO song_events (song_id,user_id,event_type,platform,duration_ms) VALUES (?,?,?,?,?)`),
songID, userID, eventType, platform, durationMs)
}

func (r *AnalyticsRepository) RecordSearchLog(userID *int, query string, filtersJSON *string, resultsCount int, platform string) {
r.db.Exec(r.db.Rebind(`INSERT INTO search_logs (user_id,query,filters,results_count,platform) VALUES (?,?,?,?,?)`),
userID, query, filtersJSON, resultsCount, platform)
}

func (r *AnalyticsRepository) RecordSession(userID *int, platform, appVersion, deviceOS string) {
r.db.Exec(r.db.Rebind(`INSERT INTO app_sessions (user_id,platform,app_version,device_os) VALUES (?,?,?,?)`),
userID, platform, nullStr(appVersion), nullStr(deviceOS))
}

func (r *AnalyticsRepository) RecordPerformance(userID *int, platform, metricType string, endpoint, screenName *string, durationMs, statusCode *int, appVersion, deviceOS string) {
r.db.Exec(r.db.Rebind(`INSERT INTO performance_logs (user_id,platform,metric_type,endpoint,screen_name,duration_ms,status_code,app_version,device_os) VALUES (?,?,?,?,?,?,?,?,?)`),
userID, platform, metricType, endpoint, screenName, durationMs, statusCode, nullStr(appVersion), nullStr(deviceOS))
}

// ── Read Aggregates ──────────────────────────────────────────────────────────

type TopSongRow struct {
SongID    *int   `db:"song_id"`
EventType string `db:"event_type"`
Count     int    `db:"count"`
}

func (r *AnalyticsRepository) TopSongs(days int, limit int) ([]TopSongRow, error) {
rows := []TopSongRow{}
err := r.db.Select(&rows, r.db.Rebind(`
SELECT song_id, event_type, COUNT(*) as count
FROM song_events
WHERE song_id IS NOT NULL AND "createdAt" >= NOW() - (? || ' days')::interval
GROUP BY song_id, event_type
ORDER BY count DESC
LIMIT ?`), days, limit)
return rows, err
}

type DailyCountRow struct {
Date  string `db:"date"`
Count int    `db:"count"`
}

func (r *AnalyticsRepository) NewUsersPerDay(days int) ([]DailyCountRow, error) {
rows := []DailyCountRow{}
err := r.db.Select(&rows, r.db.Rebind(`
SELECT DATE("createdAt")::text as date, COUNT(*) as count
FROM users
WHERE "createdAt" >= NOW() - (? || ' days')::interval
GROUP BY DATE("createdAt")
ORDER BY date ASC`), days)
return rows, err
}

func (r *AnalyticsRepository) DAU(days int) ([]DailyCountRow, error) {
rows := []DailyCountRow{}
err := r.db.Select(&rows, r.db.Rebind(`
SELECT DATE("createdAt")::text as date, COUNT(DISTINCT user_id) as count
FROM app_sessions
WHERE user_id IS NOT NULL AND "createdAt" >= NOW() - (? || ' days')::interval
GROUP BY DATE("createdAt")
ORDER BY date ASC`), days)
return rows, err
}

type TopSearchRow struct {
Query        string `db:"query"`
Count        int    `db:"count"`
AvgResults   int    `db:"avg_results"`
ZeroResults  int    `db:"zero_results"`
}

func (r *AnalyticsRepository) TopSearches(days, limit int) ([]TopSearchRow, error) {
rows := []TopSearchRow{}
err := r.db.Select(&rows, r.db.Rebind(`
SELECT query,
       COUNT(*) as count,
       ROUND(AVG(results_count))::int as avg_results,
       SUM(CASE WHEN results_count = 0 THEN 1 ELSE 0 END) as zero_results
FROM search_logs
WHERE "createdAt" >= NOW() - (? || ' days')::interval
GROUP BY query
ORDER BY count DESC
LIMIT ?`), days, limit)
return rows, err
}

type PlatformBreakdownRow struct {
Platform string `db:"platform"`
Count    int    `db:"count"`
}

func (r *AnalyticsRepository) SessionsByPlatform(days int) ([]PlatformBreakdownRow, error) {
rows := []PlatformBreakdownRow{}
err := r.db.Select(&rows, r.db.Rebind(`
SELECT platform, COUNT(*) as count
FROM app_sessions
WHERE "createdAt" >= NOW() - (? || ' days')::interval
GROUP BY platform
ORDER BY count DESC`), days)
return rows, err
}

type PerfSummaryRow struct {
MetricType string  `db:"metric_type"`
Endpoint   *string `db:"endpoint"`
ScreenName *string `db:"screen_name"`
P50        int     `db:"p50"`
P95        int     `db:"p95"`
Avg        int     `db:"avg"`
Count      int     `db:"count"`
}

func (r *AnalyticsRepository) PerformanceSummary(days int) ([]PerfSummaryRow, error) {
rows := []PerfSummaryRow{}
err := r.db.Select(&rows, r.db.Rebind(`
SELECT metric_type, endpoint, screen_name,
       PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms)::int as p50,
       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms)::int as p95,
       ROUND(AVG(duration_ms))::int as avg,
       COUNT(*) as count
FROM performance_logs
WHERE "createdAt" >= NOW() - (? || ' days')::interval
GROUP BY metric_type, endpoint, screen_name
ORDER BY p95 DESC`), days)
return rows, err
}

func nullStr(s string) *string {
if s == "" {
return nil
}
return &s
}
