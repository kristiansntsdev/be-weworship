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
	status, err := h.teams.Leave(teamID, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Successfully left the team", fiber.Map{"team_id": teamID, "user_id": cl.UserID})
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
