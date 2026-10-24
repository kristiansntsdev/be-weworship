package repositories

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type AnalyticsRepository struct {
	db *sqlx.DB
}

func NewAnalyticsRepository(db *sqlx.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// ── Write Methods ────────────────────────────────────────────────────────────

func (r *AnalyticsRepository) RecordSongEvent(songID, userID *int, eventType, platform string, durationMs *int) {
	r.db.Exec(r.db.Rebind(`INSERT INTO song_events (song_id,user_id,event_type,platform,duration_ms) VALUES (?,?,?,?,?)`),
		songID, userID, eventType, platform, durationMs)
}

// RankedSongRow is a song ranked by song_events of a given type (view / add_to_playlist / …).
type RankedSongRow struct {
	ID     int            `db:"id"`
	Title  string         `db:"title"`
	Artist sql.NullString `db:"artist"`
	Key    sql.NullString `db:"key"`
	Count  int            `db:"count"`
}

// RankedSongs returns top songs by event_type. days=0 means all time.
func (r *AnalyticsRepository) RankedSongs(eventType string, days, limit int) ([]RankedSongRow, error) {
	if limit < 1 {
		limit = 5
	}
	rows := []RankedSongRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT s.id, s.title, s.artist, s.base_chord AS key, COUNT(*)::int AS count
FROM song_events e
JOIN songs s ON s.id = e.song_id
WHERE e.event_type = ?
  AND e.song_id IS NOT NULL
  AND (? = 0 OR e."createdAt" >= NOW() - (? * INTERVAL '1 day'))
GROUP BY s.id, s.title, s.artist, s.base_chord
ORDER BY count DESC, s.title ASC
LIMIT ?`), eventType, days, days, limit)
	return rows, err
}

func (r *AnalyticsRepository) RecordSearchLog(userID *int, query string, filtersJSON *string, resultsCount int, platform string) {
	r.db.Exec(r.db.Rebind(`INSERT INTO search_logs (user_id,query,filters,results_count,platform) VALUES (?,?,?,?,?)`),
		userID, query, filtersJSON, resultsCount, platform)
}

func (r *AnalyticsRepository) RecordSession(userID *int, platform, appVersion, deviceOS string) error {
	_, err := r.db.Exec(r.db.Rebind(`INSERT INTO app_sessions (user_id,platform,app_version,device_os) VALUES (?,?,?,?)`),
		userID, platform, nullStr(appVersion), nullStr(deviceOS))
	return err
}

func (r *AnalyticsRepository) RecordPerformance(userID *int, platform, metricType string, endpoint, screenName *string, durationMs, statusCode *int, appVersion, deviceOS string) error {
	_, err := r.db.Exec(r.db.Rebind(`INSERT INTO performance_logs (user_id,platform,metric_type,endpoint,screen_name,duration_ms,status_code,app_version,device_os) VALUES (?,?,?,?,?,?,?,?,?)`),
		userID, platform, metricType, endpoint, screenName, durationMs, statusCode, nullStr(appVersion), nullStr(deviceOS))
	return err
}

// ── Read Aggregates ──────────────────────────────────────────────────────────

// TopSongRow is a song ranked by total song_events (admin analytics).
type TopSongRow struct {
	SongID    int            `db:"song_id"`
	Title     string         `db:"title"`
	Artist    sql.NullString `db:"artist"`
	PlayCount int            `db:"play_count"`
}

func (r *AnalyticsRepository) TopSongs(days int, limit int) ([]TopSongRow, error) {
	rows := []TopSongRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT s.id AS song_id, s.title, s.artist, COUNT(*)::int AS play_count
FROM song_events e
JOIN songs s ON s.id = e.song_id
WHERE e.song_id IS NOT NULL AND e."createdAt" >= NOW() - ? * INTERVAL '1 day'
GROUP BY s.id, s.title, s.artist
ORDER BY play_count DESC
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
WHERE "createdAt" >= NOW() - ? * INTERVAL '1 day'
GROUP BY DATE("createdAt")
ORDER BY date ASC`), days)
	return rows, err
}

func (r *AnalyticsRepository) DAU(days int) ([]DailyCountRow, error) {
	rows := []DailyCountRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT DATE("createdAt")::text as date, COUNT(DISTINCT user_id) as count
FROM app_sessions
WHERE user_id IS NOT NULL AND "createdAt" >= NOW() - ? * INTERVAL '1 day'
GROUP BY DATE("createdAt")
ORDER BY date ASC`), days)
	return rows, err
}

type TopSearchRow struct {
	Query       string `db:"query"`
	Count       int    `db:"count"`
	AvgResults  int    `db:"avg_results"`
	ZeroResults int    `db:"zero_results"`
}

func (r *AnalyticsRepository) TopSearches(days, limit int) ([]TopSearchRow, error) {
	rows := []TopSearchRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT query,
       COUNT(*) as count,
       ROUND(AVG(results_count))::int as avg_results,
       SUM(CASE WHEN results_count = 0 THEN 1 ELSE 0 END) as zero_results
FROM search_logs
WHERE "createdAt" >= NOW() - ? * INTERVAL '1 day'
GROUP BY query
ORDER BY count DESC
LIMIT ?`), days, limit)
	return rows, err
}

// RecentSearchesByUser returns the user's most recent unique song search queries.
func (r *AnalyticsRepository) RecentSearchesByUser(userID, limit int) ([]string, error) {
	if limit < 1 {
		limit = 10
	}
	rows := []string{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT query FROM (
  SELECT DISTINCT ON (LOWER(TRIM(query))) TRIM(query) AS query, "createdAt"
  FROM search_logs
  WHERE user_id = ? AND TRIM(query) <> ''
  ORDER BY LOWER(TRIM(query)), "createdAt" DESC
) t
ORDER BY "createdAt" DESC
LIMIT ?`), userID, limit)
	return rows, err
}

// RecentViewedSongIDs returns distinct songs the user opened, newest first.
func (r *AnalyticsRepository) RecentViewedSongIDs(userID, limit int) ([]int, error) {
	if limit < 1 {
		limit = 10
	}
	ids := []int{}
	err := r.db.Select(&ids, r.db.Rebind(`
SELECT song_id FROM (
  SELECT DISTINCT ON (song_id) song_id, "createdAt"
  FROM song_events
  WHERE user_id = ? AND event_type = 'view' AND song_id IS NOT NULL
  ORDER BY song_id, "createdAt" DESC
) t
ORDER BY "createdAt" DESC
LIMIT ?`), userID, limit)
	return ids, err
}

type PlatformBreakdownRow struct {
	Date     string `db:"date" json:"date"`
	Platform string `db:"platform" json:"platform"`
	Count    int    `db:"count" json:"count"`
}

func (r *AnalyticsRepository) SessionsByPlatform(days int) ([]PlatformBreakdownRow, error) {
	rows := []PlatformBreakdownRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT DATE("createdAt")::text as date, platform, COUNT(*) as count
FROM app_sessions
WHERE "createdAt" >= NOW() - ? * INTERVAL '1 day'
GROUP BY DATE("createdAt"), platform
ORDER BY date ASC, platform ASC`), days)
	return rows, err
}

type PerfSummaryRow struct {
	MetricType    string  `db:"metric_type" json:"metric_type"`
	Endpoint      *string `db:"endpoint" json:"endpoint"`
	ScreenName    *string `db:"screen_name" json:"screen_name"`
	P50Ms         int     `db:"p50_ms" json:"p50_ms"`
	P95Ms         int     `db:"p95_ms" json:"p95_ms"`
	AvgMs         int     `db:"avg_ms" json:"avg_ms"`
	TotalRequests int     `db:"total_requests" json:"total_requests"`
}

func (r *AnalyticsRepository) PerformanceSummary(days int) ([]PerfSummaryRow, error) {
	rows := []PerfSummaryRow{}
	err := r.db.Select(&rows, r.db.Rebind(`
SELECT metric_type,
       COALESCE(endpoint, screen_name, metric_type) as endpoint,
       screen_name,
       PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms)::int as p50_ms,
       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms)::int as p95_ms,
       ROUND(AVG(duration_ms))::int as avg_ms,
       COUNT(*) as total_requests
FROM performance_logs
WHERE "createdAt" >= NOW() - ? * INTERVAL '1 day'
GROUP BY metric_type, endpoint, screen_name
ORDER BY p95_ms DESC`), days)
	return rows, err
}

func (r *AnalyticsRepository) TotalUsers() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r *AnalyticsRepository) DAUToday() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM app_sessions WHERE user_id IS NOT NULL AND DATE("createdAt") = CURRENT_DATE`).Scan(&count)
	return count, err
}

func (r *AnalyticsRepository) MAU() (int, error) {
	var count int
	err := r.db.QueryRow(r.db.Rebind(`SELECT COUNT(DISTINCT user_id) FROM app_sessions WHERE user_id IS NOT NULL AND "createdAt" >= NOW() - 30 * INTERVAL '1 day'`)).Scan(&count)
	return count, err
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
