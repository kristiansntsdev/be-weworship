package handlers

import (
	"strings"

	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RequestAccountDeletion(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid request body")
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return utils.Fail(c, 400, "Email is required")
	}

	ip := c.IP()
	if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.SplitN(forwarded, ",", 2)[0]
	}

	if err := h.users.RequestDeletion(req.Email, ip); err != nil {
		return utils.FailErr(c, 500, "Failed to submit deletion request", err)
	}
	return utils.OK(c, 200, "Your deletion request has been received. Your account and data will be removed within 30 days.", nil)
}

func (h *Handler) GetUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	search := c.Query("search")
	rows, total, err := h.users.List(search, page, limit)
	if err != nil {
		return utils.FailErr(c, 500, "Failed to retrieve users", err)
	}
	data := make([]fiber.Map, len(rows))
	for i, u := range rows {
		data[i] = fiber.Map{
			"id":        u.ID,
			"username":  u.Username,
			"email":     u.Email,
			"role":      u.Role,
			"provider":  u.Provider,
			"verified":  u.Verified,
			"status":    u.Status,
			"createdAt": u.CreatedAt,
		}
	}
	return c.JSON(fiber.Map{
		"code":    200,
		"message": "User list retrieved successfully",
		"data":    data,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *Handler) UpdateUserRole(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, err.Error())
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid request body")
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	u, status, err := h.users.UpdateRole(id, role)
	if err != nil {
		return utils.FailErr(c, status, err.Error(), err)
	}
	return utils.OK(c, 200, "User role updated successfully", fiber.Map{
		"id":        u.ID,
		"username":  u.Username,
		"email":     u.Email,
		"role":      u.Role,
		"provider":  u.Provider,
		"verified":  u.Verified,
		"status":    u.Status,
		"createdAt": u.CreatedAt,
	})
}
