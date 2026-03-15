package handlers

import (
"be-songbanks-v1/api/middleware"
"be-songbanks-v1/api/utils"
"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterUser(c *fiber.Ctx) error {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	data, status, err := h.auth.Register(req.Name, req.Email, req.Password)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	injectDetail(h, data)
	return utils.OK(c, status, "Account created successfully", data)
}

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
injectDetail(h, data)
return utils.OK(c, 200, "Login successful", data)
}

// injectDetail fetches user_detail and embeds it into the "user" map returned by auth responses.
func injectDetail(h *Handler, data map[string]any) {
	userMap, ok := data["user"].(map[string]any)
	if !ok {
		return
	}
	userID, ok := userMap["id"].(int)
	if !ok {
		return
	}
	detail, _ := h.users.GetDetail(userID)
	detailMap := map[string]any{"full_name": nil, "province": nil, "city": nil, "postal_code": nil}
	if detail != nil {
		if detail.FullName.Valid {
			detailMap["full_name"] = detail.FullName.String
		}
		if detail.Province.Valid {
			detailMap["province"] = detail.Province.String
		}
		if detail.City.Valid {
			detailMap["city"] = detail.City.String
		}
		if detail.PostalCode.Valid {
			detailMap["postal_code"] = detail.PostalCode.String
		}
	}
	userMap["detail"] = detailMap
}

// GoogleLogin redirects the browser to Google's OAuth consent screen.
// Accepts ?client=mobile or ?state=mobile (mobile app uses ?state=mobile).
// Mobile may pass ?redirect_uri=<encoded> to override deep-link scheme (e.g. Expo Go uses exp://).
func (h *Handler) GoogleLogin(c *fiber.Ctx) error {
client := c.Query("client", "")
if client == "" {
client = c.Query("state", "web")
}
if client != "web" && client != "mobile" {
client = "web"
}
state := client
if client == "mobile" {
if override := c.Query("redirect_uri", ""); override != "" {
state = "mobile|" + override
}
}
return c.Redirect(h.auth.GoogleAuthURL(state), fiber.StatusFound)
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
detail, _ := h.users.GetDetail(cl.UserID)
detailMap := map[string]any{"full_name": nil, "province": nil, "city": nil, "postal_code": nil}
if detail != nil {
if detail.FullName.Valid {
detailMap["full_name"] = detail.FullName.String
}
if detail.Province.Valid {
detailMap["province"] = detail.Province.String
}
if detail.City.Valid {
detailMap["city"] = detail.City.String
}
if detail.PostalCode.Valid {
detailMap["postal_code"] = detail.PostalCode.String
}
}
avatarURL, _ := h.users.GetAvatarURL(cl.UserID)
var avatarAny any
if avatarURL != "" {
avatarAny = avatarURL
}
return utils.OK(c, 200, "Current user retrieved successfully", fiber.Map{
"user": fiber.Map{
"id":         cl.UserID,
"name":       cl.Name,
"email":      cl.Email,
"role":       cl.Role,
"avatar_url": avatarAny,
"detail":     detailMap,
},
})
}

func (h *Handler) UpdateAvatar(c *fiber.Ctx) error {
cl := middleware.GetClaims(c)
var req struct {
AvatarURL string `json:"avatar_url"`
}
if err := c.BodyParser(&req); err != nil {
return utils.Fail(c, 400, "Invalid request body")
}
if req.AvatarURL == "" {
return utils.Fail(c, 400, "avatar_url is required")
}
if err := h.users.UpdateAvatarURL(cl.UserID, req.AvatarURL); err != nil {
return utils.Fail(c, 500, "Failed to update avatar")
}
return utils.OK(c, 200, "Avatar updated successfully", nil)
}

func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
cl := middleware.GetClaims(c)
var req struct {
FullName   *string `json:"full_name"`
Province   *string `json:"province"`
City       *string `json:"city"`
PostalCode *string `json:"postal_code"`
}
if err := c.BodyParser(&req); err != nil {
return utils.Fail(c, 400, "Invalid request body")
}
if err := h.users.UpdateProfile(cl.UserID, req.FullName, req.Province, req.City, req.PostalCode); err != nil {
return utils.Fail(c, 500, "Failed to update profile")
}
return utils.OK(c, 200, "Profile updated successfully", nil)
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
"isMaintainer":  cl.Role == "admin" || cl.Role == "maintainer",
})
}
