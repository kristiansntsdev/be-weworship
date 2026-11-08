package handlers

import (
	"errors"

	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/services"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

// GetMixes returns the "Mixed for you" shelf: personal mixes (rebuilt at most daily) and the mood chips.
func (h *Handler) GetMixes(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	data, err := h.mixes.Mixes(c.UserContext(), cl.UserID)
	if err != nil {
		return utils.FailErr(c, 500, "Failed to build mixes", err)
	}
	return utils.OK(c, 200, "Mixes retrieved", data)
}

// GetMoodMix returns the mix for one mood chip (e.g. /mixes/moods/calm).
func (h *Handler) GetMoodMix(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	mix, err := h.mixes.MoodMix(c.UserContext(), cl.UserID, c.Params("mood"))
	switch {
	case errors.Is(err, services.ErrMoodNotFound):
		return utils.Fail(c, 404, "Mood not found")
	case errors.Is(err, services.ErrMixesUnavailable):
		return utils.Fail(c, 503, "Mixes are not available right now")
	case err != nil:
		return utils.FailErr(c, 500, "Failed to build mix", err)
	}
	return utils.OK(c, 200, "Mood mix retrieved", mix)
}
