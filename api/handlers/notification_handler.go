package handlers

import (
	"strconv"

	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"

	"github.com/gofiber/fiber/v2"
)

// RegisterDeviceToken saves a device token for the authenticated user.
// POST /api/notifications/device-token
// Body: { "token": "...", "platform": "android" | "ios" }
func (h *Handler) RegisterDeviceToken(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}

	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := c.BodyParser(&req); err != nil || req.Token == "" {
		return utils.Fail(c, 400, "token is required")
	}
	if req.Platform != "android" && req.Platform != "ios" {
		req.Platform = "android"
	}

	if err := h.notifications.SaveDeviceToken(cl.UserID, req.Token, req.Platform); err != nil {
		return utils.Fail(c, 500, "Failed to save device token")
	}
	return utils.OK(c, 200, "Device token registered", nil)
}

// UnregisterDeviceToken removes a device token for the authenticated user (logout).
// DELETE /api/notifications/device-token
// Body: { "token": "..." }
func (h *Handler) UnregisterDeviceToken(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil || req.Token == "" {
		return utils.Fail(c, 400, "token is required")
	}

	if err := h.notifications.RemoveDeviceToken(cl.UserID, req.Token); err != nil {
		return utils.Fail(c, 500, "Failed to remove device token")
	}
	return utils.OK(c, 200, "Device token removed", nil)
}

// GetNotifications returns the notification inbox for the authenticated user.
// Includes both targeted notifications and broadcast rows (new song, etc.).
// GET /api/notifications?page=1&limit=20
func (h *Handler) GetNotifications(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit > 50 {
		limit = 50
	}
	rows, err := h.notifications.GetNotifications(cl.UserID, page, limit)
	if err != nil {
		return utils.Fail(c, 500, "Failed to fetch notifications")
	}
	return utils.OK(c, 200, "OK", rows)
}

// GetUnreadCount returns the number of unread targeted notifications.
// Broadcast rows (new song) are excluded from this count.
// GET /api/notifications/unread-count
func (h *Handler) GetUnreadCount(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	count, err := h.notifications.GetUnreadCount(cl.UserID)
	if err != nil {
		return utils.Fail(c, 500, "Failed to fetch unread count")
	}
	return utils.OK(c, 200, "OK", fiber.Map{"unread_count": count})
}

// MarkNotificationRead marks a single targeted notification as read.
// POST /api/notifications/:id/read
func (h *Handler) MarkNotificationRead(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return utils.Fail(c, 400, "invalid notification id")
	}
	if err := h.notifications.MarkAsRead(id, cl.UserID); err != nil {
		return utils.Fail(c, 500, "Failed to mark notification as read")
	}
	return utils.OK(c, 200, "Notification marked as read", nil)
}

// GetFCMStatus returns whether the FCM provider is active (admin-only debug endpoint).
// GET /api/admin/fcm-status
func (h *Handler) GetFCMStatus(c *fiber.Ctx) error {
	enabled, projectID := h.notifications.FCMStatus()
	return utils.OK(c, 200, "OK", fiber.Map{
		"fcm_enabled": enabled,
		"project_id":  projectID,
	})
}
