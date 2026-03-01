package handlers

import (
"be-songbanks-v1/api/middleware"
"be-songbanks-v1/api/utils"
"github.com/gofiber/fiber/v2"
)

func (h *Handler) Login(c *fiber.Ctx) error {
var req struct {
Email    string `json:"email"`
Password string `json:"password"`
}
if err := c.BodyParser(&req); err != nil {
return utils.Fail(c, 400, "Invalid JSON")
}
data, status, err := h.auth.Login(req.Email, req.Password)
if err != nil {
return utils.Fail(c, status, err.Error())
}
return utils.OK(c, 200, "Login successful", data)
}

// GoogleLogin redirects the browser to Google's OAuth consent screen.
// Query param: client=web|mobile  (defaults to "web")
func (h *Handler) GoogleLogin(c *fiber.Ctx) error {
client := c.Query("client", "web")
if client != "web" && client != "mobile" {
client = "web"
}
return c.Redirect(h.auth.GoogleAuthURL(client), fiber.StatusFound)
}

// GoogleCallback handles the redirect back from Google after the user consents.
func (h *Handler) GoogleCallback(c *fiber.Ctx) error {
code := c.Query("code")
state := c.Query("state", "web")

if c.Query("error") != "" || code == "" {
return c.Redirect(h.auth.GoogleLoginErrorURL("google_cancelled"), fiber.StatusFound)
}

redirectURL, err := h.auth.GoogleCallback(code, state)
if err != nil {
return c.Redirect(h.auth.GoogleLoginErrorURL("google_failed"), fiber.StatusFound)
}

return c.Redirect(redirectURL, fiber.StatusFound)
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
cl := middleware.GetClaims(c)
return utils.OK(c, 200, "Current user retrieved successfully", fiber.Map{
"user": fiber.Map{
"id":    cl.UserID,
"name":  cl.Name,
"email": cl.Email,
"role":  cl.Role,
},
})
}

func (h *Handler) CheckPermission(c *fiber.Ctx) error {
cl := middleware.GetClaims(c)
role := c.Query("role")
if role != "" && cl.Role != role {
return utils.Fail(c, 403, "Access denied")
}
return utils.OK(c, 200, "Permission granted", fiber.Map{
"hasPermission": true,
"role":          cl.Role,
"isAdmin":       cl.Role == "admin",
})
}
