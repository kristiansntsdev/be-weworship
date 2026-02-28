package handlers

import (
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) NotesUnavailable(c *fiber.Ctx) error {
	return utils.Fail(c, 501, "Notes endpoints are unavailable: table `notes` is missing in provided db.schema.json")
}
