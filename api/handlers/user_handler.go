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
limit := c.QueryInt("limit", 20)
if limit < 1 {
limit = 20
}
search := c.Query("search")
rows, total, err := h.users.List(search, page, limit)
if err != nil {
return utils.Fail(c, 500, "Failed to retrieve users")
}
return c.JSON(fiber.Map{
"code":    200,
"message": "User list retrieved successfully",
"data":    rows,
"pagination": fiber.Map{
"total": total,
"page":  page,
"limit": limit,
},
"search": search,
})
}
