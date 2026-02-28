package handlers

import (
	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreatePlaylist(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	var req struct {
		PlaylistName string `json:"playlist_name"`
		Songs        []int  `json:"songs"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	data, status, err := h.playlists.Create(cl.UserID, req.PlaylistName, req.Songs)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 201, "Playlist created successfully", data)
}

func (h *Handler) GetPlaylists(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	data, pagination, err := h.playlists.List(cl.UserID, page, limit)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve playlists")
	}
	return c.JSON(fiber.Map{"code": 200, "message": "Playlists retrieved successfully", "data": data, "pagination": pagination})
}

func (h *Handler) GetPlaylistByID(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	data, status, err := h.playlists.GetByIDWithAccess(id, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Playlist retrieved successfully", data)
}

func (h *Handler) UpdatePlaylist(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	var req struct {
		PlaylistName string `json:"playlist_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	status, err := h.playlists.UpdateName(id, cl.UserID, req.PlaylistName)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Playlist updated successfully", fiber.Map{"id": id, "playlist_name": req.PlaylistName})
}

func (h *Handler) DeletePlaylist(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	status, err := h.playlists.Delete(id, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Playlist deleted successfully", fiber.Map{"id": id})
}

func (h *Handler) GenerateSharelink(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	data, status, err := h.playlists.GenerateSharelink(id, cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 201, "Sharelink generated successfully", data)
}

func (h *Handler) JoinPlaylist(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	data, status, err := h.playlists.Join(c.Params("shareToken"), cl.UserID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 201, "Successfully joined playlist team", data)
}

func (h *Handler) AddSongsToPlaylist(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	var req struct {
		SongIDs []int `json:"songIds"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	status, err := h.playlists.AddSongs(id, cl.UserID, req.SongIDs)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Song(s) added to playlist successfully", fiber.Map{"playlist_id": id, "songIds": req.SongIDs})
}

func (h *Handler) AddSongToPlaylistWithBaseChord(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	songID, err := parseID(c, "songId")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	var req struct {
		BaseChord string `json:"base_chord"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	status, err := h.playlists.AddSongWithBaseChord(id, cl.UserID, songID, req.BaseChord)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Song added to playlist with base chord successfully", fiber.Map{"playlist_id": id, "song_id": songID, "base_chord": req.BaseChord})
}

func (h *Handler) RemoveSongFromPlaylist(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid playlist ID")
	}
	songID, err := parseID(c, "songId")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	status, err := h.playlists.RemoveSong(id, cl.UserID, songID)
	if err != nil {
		return utils.Fail(c, status, err.Error())
	}
	return utils.OK(c, 200, "Song removed from playlist successfully", fiber.Map{"playlist_id": id, "song_id": songID})
}
