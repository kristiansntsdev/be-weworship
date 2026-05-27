package handlers

import (
	"strings"

	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"

	"github.com/gofiber/fiber/v2"
)

// SearchUsers returns users whose username or email matches the query (≥2 chars).
// GET /api/users/search?q=:query
func (h *Handler) SearchUsers(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		return utils.Fail(c, 400, "query must be at least 2 characters")
	}
	users, err := h.invitations.SearchUsers(q, cl.UserID)
	if err != nil {
		return utils.Fail(c, 500, "Failed to search users")
	}
	return utils.OK(c, 200, "OK", users)
}

// SendInvitation creates a playlist invitation from the authenticated leader/co-lead.
// POST /api/playlists/:id/invitations
// Body: { "invitee_id": number }
func (h *Handler) SendInvitation(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	playlistID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, err.Error())
	}
	var req struct {
		InviteeID int `json:"invitee_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.InviteeID == 0 {
		return utils.Fail(c, 400, "invitee_id is required")
	}
	invitationID, status, err := h.invitations.SendInvitation(playlistID, cl.UserID, req.InviteeID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, status, "Invitation sent", fiber.Map{"invitation_id": invitationID})
}

// AcceptInvitation accepts a pending playlist invitation.
// POST /api/invitations/:id/accept
func (h *Handler) AcceptInvitation(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, err.Error())
	}
	status, err := h.invitations.AcceptInvitation(id, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Invitation accepted", nil)
}

// DeclineInvitation declines a pending playlist invitation.
// POST /api/invitations/:id/decline
func (h *Handler) DeclineInvitation(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, err.Error())
	}
	status, err := h.invitations.DeclineInvitation(id, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Invitation declined", nil)
}
