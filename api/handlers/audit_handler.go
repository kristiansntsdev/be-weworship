package handlers

import (
	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetAuditLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	action := c.Query("action")
	entityType := c.Query("entity_type")

	var userID *int
	if v := c.QueryInt("user_id", 0); v > 0 {
		userID = &v
	}

	rows, total, err := h.audit.List(action, entityType, userID, page, limit)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve audit logs")
	}

	cl := middleware.GetClaims(c)
	_ = cl // available if needed for filtering

	return c.JSON(fiber.Map{
		"code":    200,
		"message": "Audit logs retrieved",
		"data":    rows,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}
