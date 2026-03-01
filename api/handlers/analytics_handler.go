package handlers

import (
"be-songbanks-v1/api/middleware"
"be-songbanks-v1/api/utils"
"github.com/gofiber/fiber/v2"
)

// ── Admin read endpoints ─────────────────────────────────────────────────────

func (h *Handler) GetAnalyticsSongs(c *fiber.Ctx) error {
days := c.QueryInt("days", 30)
limit := c.QueryInt("limit", 20)
data, err := h.analytics.TopSongs(days, limit)
if err != nil {
return utils.Fail(c, 500, "Failed to retrieve song analytics")
}
return utils.OK(c, 200, "Song analytics retrieved", data)
}

func (h *Handler) GetAnalyticsUsers(c *fiber.Ctx) error {
days := c.QueryInt("days", 30)
data, err := h.analytics.UserStats(days)
if err != nil {
return utils.Fail(c, 500, "Failed to retrieve user analytics")
}
return utils.OK(c, 200, "User analytics retrieved", data)
}

func (h *Handler) GetAnalyticsSearches(c *fiber.Ctx) error {
days := c.QueryInt("days", 30)
limit := c.QueryInt("limit", 20)
data, err := h.analytics.TopSearches(days, limit)
if err != nil {
return utils.Fail(c, 500, "Failed to retrieve search analytics")
}
return utils.OK(c, 200, "Search analytics retrieved", data)
}

func (h *Handler) GetAnalyticsSessions(c *fiber.Ctx) error {
days := c.QueryInt("days", 30)
data, err := h.analytics.SessionsByPlatform(days)
if err != nil {
return utils.Fail(c, 500, "Failed to retrieve session analytics")
}
return utils.OK(c, 200, "Session analytics retrieved", data)
}

func (h *Handler) GetAnalyticsPerformance(c *fiber.Ctx) error {
days := c.QueryInt("days", 7)
data, err := h.analytics.PerformanceSummary(days)
if err != nil {
return utils.Fail(c, 500, "Failed to retrieve performance analytics")
}
return utils.OK(c, 200, "Performance analytics retrieved", data)
}

// ── Public write endpoint (mobile pushes client-side metrics) ────────────────

func (h *Handler) RecordPerformance(c *fiber.Ctx) error {
var req struct {
MetricType string  `json:"metric_type"`
ScreenName *string `json:"screen_name"`
DurationMs int     `json:"duration_ms"`
Platform   string  `json:"platform"`
AppVersion string  `json:"app_version"`
DeviceOS   string  `json:"device_os"`
}
if err := c.BodyParser(&req); err != nil || req.MetricType == "" || req.DurationMs <= 0 {
return utils.Fail(c, 400, "metric_type and duration_ms are required")
}

cl := middleware.GetClaims(c)
var userID *int
if cl != nil {
id := cl.UserID
userID = &id
}

h.analytics.RecordPerformance(userID, req.Platform, req.MetricType, nil, req.ScreenName, &req.DurationMs, nil, req.AppVersion, req.DeviceOS)
return utils.OK(c, 200, "Performance recorded", nil)
}

// RecordSession is called on app open so the backend can compute DAU/MAU.
func (h *Handler) RecordSession(c *fiber.Ctx) error {
var req struct {
Platform   string `json:"platform"`
AppVersion string `json:"app_version"`
DeviceOS   string `json:"device_os"`
}
if err := c.BodyParser(&req); err != nil {
return utils.Fail(c, 400, "Invalid JSON")
}
cl := middleware.GetClaims(c)
var userID *int
if cl != nil {
id := cl.UserID
userID = &id
}
h.analytics.RecordSession(userID, req.Platform, req.AppVersion, req.DeviceOS)
return utils.OK(c, 200, "Session recorded", nil)
}
