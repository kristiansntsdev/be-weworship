package handlers

import (
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
