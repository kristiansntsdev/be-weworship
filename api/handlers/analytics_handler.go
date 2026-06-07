package handlers

import (
	"strings"

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
		return utils.FailErr(c, 500, "Failed to retrieve song analytics", err)
	}
	return utils.OK(c, 200, "Song analytics retrieved", data)
}

func (h *Handler) GetAnalyticsUsers(c *fiber.Ctx) error {
	days := c.QueryInt("days", 30)
	data, err := h.analytics.UserStats(days)
	if err != nil {
		return utils.FailErr(c, 500, "Failed to retrieve user analytics", err)
	}
	return utils.OK(c, 200, "User analytics retrieved", data)
}

func (h *Handler) GetAnalyticsSearches(c *fiber.Ctx) error {
	days := c.QueryInt("days", 30)
	limit := c.QueryInt("limit", 20)
	data, err := h.analytics.TopSearches(days, limit)
	if err != nil {
		return utils.FailErr(c, 500, "Failed to retrieve search analytics", err)
	}
	return utils.OK(c, 200, "Search analytics retrieved", data)
}

func (h *Handler) GetAnalyticsSessions(c *fiber.Ctx) error {
	days := c.QueryInt("days", 30)
	data, err := h.analytics.SessionsByPlatform(days)
	if err != nil {
		return utils.FailErr(c, 500, "Failed to retrieve session analytics", err)
	}
	return utils.OK(c, 200, "Session analytics retrieved", data)
}

func (h *Handler) GetAnalyticsPerformance(c *fiber.Ctx) error {
	days := c.QueryInt("days", 7)
	data, err := h.analytics.PerformanceSummary(days)
	if err != nil {
		return utils.FailErr(c, 500, "Failed to retrieve performance analytics", err)
	}
	return utils.OK(c, 200, "Performance analytics retrieved", data)
}

// ── Public write endpoint (mobile pushes client-side metrics) ────────────────

func (h *Handler) RecordPerformance(c *fiber.Ctx) error {
	var req struct {
		MetricType string  `json:"metric_type"`
		Endpoint   *string `json:"endpoint"`
		ScreenName *string `json:"screen_name"`
		DurationMs int     `json:"duration_ms"`
		StatusCode *int    `json:"status_code"`
		Platform   string  `json:"platform"`
		AppVersion string  `json:"app_version"`
		DeviceOS   string  `json:"device_os"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.MetricType) == "" || req.DurationMs <= 0 {
		return utils.Fail(c, 400, "metric_type and duration_ms are required")
	}
	req.MetricType = strings.TrimSpace(req.MetricType)
	req.Endpoint = normalizeOptionalString(req.Endpoint)
	req.ScreenName = normalizeOptionalString(req.ScreenName)
	req.Platform = normalizeAnalyticsPlatform(req.Platform)

	cl := middleware.GetClaims(c)
	var userID *int
	if cl != nil {
		id := cl.UserID
		userID = &id
	}

	if err := h.analytics.RecordPerformance(userID, req.Platform, req.MetricType, req.Endpoint, req.ScreenName, &req.DurationMs, req.StatusCode, req.AppVersion, req.DeviceOS); err != nil {
		return utils.FailErr(c, 500, "Failed to record performance", err)
	}
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
	req.Platform = normalizeAnalyticsPlatform(req.Platform)
	if err := h.analytics.RecordSession(userID, req.Platform, req.AppVersion, req.DeviceOS); err != nil {
		return utils.FailErr(c, 500, "Failed to record session", err)
	}
	return utils.OK(c, 200, "Session recorded", nil)
}

func normalizeAnalyticsPlatform(platform string) string {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return "unknown"
	}
	return platform
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
