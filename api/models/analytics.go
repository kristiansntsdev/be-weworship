package models

import "time"

type SongEvent struct {
ID         int       `db:"id"`
SongID     *int      `db:"song_id"`
UserID     *int      `db:"user_id"`
EventType  string    `db:"event_type"`
Platform   string    `db:"platform"`
DurationMs *int      `db:"duration_ms"`
CreatedAt  time.Time `db:"createdAt"`
}

type SearchLog struct {
ID           int       `db:"id"`
UserID       *int      `db:"user_id"`
Query        string    `db:"query"`
Filters      *string   `db:"filters"`
ResultsCount int       `db:"results_count"`
Platform     string    `db:"platform"`
CreatedAt    time.Time `db:"createdAt"`
}

type AppSession struct {
ID         int       `db:"id"`
UserID     *int      `db:"user_id"`
Platform   string    `db:"platform"`
AppVersion *string   `db:"app_version"`
DeviceOS   *string   `db:"device_os"`
CreatedAt  time.Time `db:"createdAt"`
}

type PerformanceLog struct {
ID         int       `db:"id"`
UserID     *int      `db:"user_id"`
Platform   string    `db:"platform"`
MetricType string    `db:"metric_type"`
Endpoint   *string   `db:"endpoint"`
ScreenName *string   `db:"screen_name"`
DurationMs int       `db:"duration_ms"`
StatusCode *int      `db:"status_code"`
AppVersion *string   `db:"app_version"`
DeviceOS   *string   `db:"device_os"`
CreatedAt  time.Time `db:"createdAt"`
}
