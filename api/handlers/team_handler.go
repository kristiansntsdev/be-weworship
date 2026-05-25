package handlers

import (
	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetMyTeams(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	rows, err := h.teams.ListByLead(cl.UserID)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve playlist teams")
	}
	return utils.OK(c, 200, "Playlist teams retrieved successfully", rows)
}

func (h *Handler) GetTeamByID(c *fiber.Ctx) error {
	teamID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid team ID")
	}
	data, status, err := h.teams.GetByID(teamID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Playlist team details retrieved successfully", data)
}

func (h *Handler) RemoveMember(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	teamID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid team ID")
	}
	memberID, err := parseID(c, "user_id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid user ID")
	}
	status, err := h.teams.RemoveMember(teamID, memberID, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Member removed from team successfully", fiber.Map{"team_id": teamID, "user_id": memberID})
}

func (h *Handler) LeaveTeam(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	teamID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid team ID")
	}
	// Get owner info before leaving (team still exists at this point)
	ownerID, playlistName, ownerErr := h.playlists.GetOwnerByTeamID(teamID)
	status, err := h.teams.Leave(teamID, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	// Notify the playlist owner that a member left
	if ownerErr == nil && ownerID != cl.UserID {
		memberName := cl.Username
		if memberName == "" {
			memberName = cl.Email
		}
		h.notifications.NotifyMemberLeft(playlistName, memberName, ownerID)
	}
	return utils.OK(c, 200, "Successfully left the team", fiber.Map{"team_id": teamID, "user_id": cl.UserID})
}

func (h *Handler) PromoteCoLead(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	teamID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid team ID")
	}
	var req struct {
		UserID int `json:"user_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.UserID == 0 {
		return utils.Fail(c, 400, "user_id is required")
	}
	status, err := h.teams.PromoteToCoLead(teamID, cl.UserID, req.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Member promoted to co-lead successfully", fiber.Map{"team_id": teamID, "user_id": req.UserID})
}

func (h *Handler) DemoteCoLead(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	teamID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid team ID")
	}
	coLeadID, err := parseID(c, "user_id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid user ID")
	}
	status, err := h.teams.DemoteCoLead(teamID, cl.UserID, coLeadID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Co-lead demoted successfully", fiber.Map{"team_id": teamID, "user_id": coLeadID})
}

func (h *Handler) DeleteTeam(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	teamID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid team ID")
	}
	status, err := h.teams.Delete(teamID, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Playlist team deleted successfully", fiber.Map{"id": teamID})
}

// ── Playlist member roles ─────────────────────────────────────────────────────

// GetPlaylistMembers handles GET /api/playlists/:id/members
func (h *Handler) GetPlaylistMembers(c *fiber.Ctx) error {
	playlistID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	data, status, err := h.teams.ListMembers(playlistID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Playlist members retrieved successfully", data)
}

// SetPlaylistMemberRole handles PUT /api/playlists/:id/members/:userId/role
func (h *Handler) SetPlaylistMemberRole(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	playlistID, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	targetUserID, err := parseID(c, "userId")
	if err != nil {
		return utils.Fail(c, 400, "Invalid user ID")
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil || req.Role == "" {
		return utils.Fail(c, 400, "role is required")
	}
	status, err := h.teams.SetMemberRole(playlistID, cl.UserID, targetUserID, req.Role)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Member role updated successfully", fiber.Map{
		"playlist_id": playlistID,
		"user_id":     targetUserID,
		"role":        req.Role,
	})
}
