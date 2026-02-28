package handlers

import (
	"be-songbanks-v1/api/utils"
	"bytes"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetArtists(c *fiber.Ctx) error {
	artists, err := h.songs.Artists()
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve artists")
	}
	return utils.OK(c, 200, "Artists retrieved successfully", artists)
}

func (h *Handler) GetSongs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	// Default to all songs when limit is not provided.
	limit := c.QueryInt("limit", 100000)
	if limit < 1 {
		limit = 100000
	}
	if limit > 1000 {
		limit = 1000
	}
	data, pagination, err := h.songs.List(page, limit, c.Query("search"), c.Query("base_chord"), c.Query("sortBy", "createdAt"), c.Query("sortOrder", "DESC"), utils.ParseCSVInts(c.Query("tag_ids")))
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve songs")
	}
	return c.JSON(fiber.Map{"code": 200, "message": "Songs retrieved successfully", "data": data, "pagination": pagination})
}

func (h *Handler) GetSongByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	data, found, err := h.songs.GetByID(id)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve song")
	}
	if !found {
		return utils.Fail(c, 404, "Song not found")
	}
	return utils.OK(c, 200, "Song details retrieved successfully", data)
}

func (h *Handler) CreateSong(c *fiber.Ctx) error {
	var req struct {
		Title          string   `json:"title"`
		Artist         any      `json:"artist"`
		BaseChord      *string  `json:"base_chord"`
		LyricsAndChord *string  `json:"lyrics_and_chords"`
		TagNames       []string `json:"tag_names"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	if req.Title == "" || req.Artist == nil {
		return utils.Fail(c, 400, "title and artist are required")
	}
	out, err := h.songs.Create(req.Title, req.Artist, req.BaseChord, req.LyricsAndChord, req.TagNames)
	if err != nil {
		return utils.Fail(c, 500, "Failed to create song")
	}
	return utils.OK(c, 201, "Song created successfully", out)
}

func (h *Handler) UpdateSong(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	var req struct {
		Title          *string  `json:"title"`
		Artist         any      `json:"artist"`
		BaseChord      *string  `json:"base_chord"`
		LyricsAndChord *string  `json:"lyrics_and_chords"`
		TagNames       []string `json:"tag_names"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	hasTagNames := bytes.Contains(c.Body(), []byte(`"tag_names"`))
	ok, err := h.songs.Update(id, req.Title, req.Artist, req.BaseChord, req.LyricsAndChord, req.TagNames, hasTagNames)
	if err != nil {
		return utils.Fail(c, 500, "Failed to update song")
	}
	if !ok {
		return utils.Fail(c, 404, "Song not found")
	}
	return utils.OK(c, 200, "Song updated successfully", fiber.Map{"id": id})
}

func (h *Handler) DeleteSong(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	ok, err := h.songs.Delete(id)
	if err != nil {
		return utils.Fail(c, 500, "Failed to delete song")
	}
	if !ok {
		return utils.Fail(c, 404, "Song not found")
	}
	return utils.OK(c, 200, "Song deleted successfully", fiber.Map{"id": id})
}
