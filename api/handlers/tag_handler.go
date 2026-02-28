package handlers

import (
	"strings"

	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetTags(c *fiber.Ctx) error {
	rows, err := h.tags.List(c.Query("search"))
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve tags")
	}
	return utils.OK(c, 200, "Tags retrieved successfully", rows)
}

func (h *Handler) GetOrCreateTag(c *fiber.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	if strings.TrimSpace(req.Name) == "" {
		return utils.Fail(c, 400, "name is required")
	}
	tag, created, err := h.tags.GetOrCreate(req.Name)
	if err != nil {
		return utils.Fail(c, 500, "Failed to get/create tag")
	}
	msg := "Tag retrieved successfully"
	if created {
		msg = "Tag created successfully"
	}
	return utils.OK(c, 200, msg, map[string]any{"tag": tag, "created": created})
}
