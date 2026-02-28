package handlers

import (
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	search := c.Query("search")
	rows, pagination, err := h.users.List(search, page, limit)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve users")
	}
	return c.JSON(fiber.Map{"code": 200, "message": "User list retrieved successfully", "data": rows, "pagination": pagination, "search": search})
}
