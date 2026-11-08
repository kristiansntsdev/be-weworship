package handlers

import (
	"errors"

	"be-songbanks-v1/api/services"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

// SearchProfiles finds people by username or display name (GET /profiles?q=). Never returns emails.
func (h *Handler) SearchProfiles(c *fiber.Ctx) error {
	data, err := h.profiles.Search(c.Query("q"))
	if err != nil {
		return utils.FailErr(c, 500, "Failed to search people", err)
	}
	return utils.OK(c, 200, "People retrieved", data)
}

// GetProfile returns a user's public profile and public playlists (GET /profiles/:username).
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	data, err := h.profiles.Get(c.Params("username"))
	if errors.Is(err, services.ErrProfileNotFound) {
		return utils.Fail(c, 404, "User not found")
	}
	if err != nil {
		return utils.FailErr(c, 500, "Failed to retrieve profile", err)
	}
	return utils.OK(c, 200, "Profile retrieved", data)
}
