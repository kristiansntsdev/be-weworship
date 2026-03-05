package handlers

import (
	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"
	"bytes"
	"strings"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetHome(c *fiber.Ctx) error {
	stats, err := h.songs.HomeStats()
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve home stats")
	}
	return utils.OK(c, 200, "Home stats retrieved successfully", stats)
}

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
	// Default to all songs when limit is not provided (mobile app compatibility).
	limit := c.QueryInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	var hasLink *bool
	if v := c.Query("has_link"); v == "true" {
		t := true; hasLink = &t
	} else if v == "false" {
		f := false; hasLink = &f
	}
	var chordPro *bool
	if v := c.Query("chordpro"); v == "true" {
		t := true; chordPro = &t
	} else if v == "false" {
		f := false; chordPro = &f
	}
	data, pagination, err := h.songs.List(page, limit, c.Query("search"), c.Query("base_chord"), c.Query("sortBy", "createdAt"), c.Query("sortOrder", "DESC"), utils.ParseCSVInts(c.Query("tag_ids")), hasLink, chordPro, strings.Contains(c.Get("User-Agent"), "okhttp"))
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve songs")
	}
	return c.JSON(fiber.Map{"code": 200, "message": "Songs retrieved successfully", "data": data, "pagination": pagination})
}

func (h *Handler) GetSongByID(c *fiber.Ctx) error {
	identifier := c.Params("id")
	// Try numeric ID first, fall back to slug lookup
	if numID, err := parseInt(identifier); err == nil {
		data, found, err := h.songs.GetByID(numID)
		if err != nil {
			return utils.Fail(c, 500, "Failed to retrieve song")
		}
		if !found {
			return utils.Fail(c, 404, "Song not found")
		}
		return utils.OK(c, 200, "Song details retrieved successfully", data)
	}
	data, found, err := h.songs.GetBySlug(identifier)
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
		Title           string   `json:"title"`
		Artist          any      `json:"artist"`
		BaseChord       *string  `json:"base_chord"`
		Bpm             *int     `json:"bpm"`
		LyricsAndChord  *string  `json:"lyrics_and_chords"`
		ExternalLinks   *string  `json:"external_links"`
		DmcaTakedown    bool     `json:"dmca_takedown"`
		DmcaStatusNotes *string  `json:"dmca_status_notes"`
		TagNames        []string `json:"tag_names"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	if req.Title == "" || req.Artist == nil {
		return utils.Fail(c, 400, "title and artist are required")
	}
	out, err := h.songs.Create(req.Title, req.Artist, req.BaseChord, req.Bpm, req.LyricsAndChord, req.ExternalLinks, req.DmcaTakedown, req.DmcaStatusNotes, req.TagNames)
	if err != nil {
		return utils.Fail(c, 500, "Failed to create song")
	}
	cl := middleware.GetClaims(c)
	if cl != nil {
		uid := cl.UserID
		h.audit.Log(&uid, cl.Name, cl.Email, "create", "song", nil, req.Title, map[string]any{"title": req.Title, "artist": req.Artist, "base_chord": req.BaseChord})
	}
	return utils.OK(c, 201, "Song created successfully", out)
}

func (h *Handler) UpdateSong(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	var req struct {
		Title           *string  `json:"title"`
		Artist          any      `json:"artist"`
		BaseChord       *string  `json:"base_chord"`
		Bpm             *int     `json:"bpm"`
		LyricsAndChord  *string  `json:"lyrics_and_chords"`
		ExternalLinks   *string  `json:"external_links"`
		DmcaTakedown    *bool    `json:"dmca_takedown"`
		DmcaStatusNotes *string  `json:"dmca_status_notes"`
		TagNames        []string `json:"tag_names"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	hasTagNames := bytes.Contains(c.Body(), []byte(`"tag_names"`))

	// Snapshot before for diff
	beforeMap, _, _ := h.songs.GetByID(id)

	ok, err := h.songs.Update(id, req.Title, req.Artist, req.BaseChord, req.Bpm, req.LyricsAndChord, req.ExternalLinks, req.DmcaTakedown, req.DmcaStatusNotes, req.TagNames, hasTagNames)
	if err != nil {
		return utils.Fail(c, 500, "Failed to update song")
	}
	if !ok {
		return utils.Fail(c, 404, "Song not found")
	}
	cl := middleware.GetClaims(c)
	if cl != nil {
		uid := cl.UserID
		changes := map[string]any{}
		if req.Title != nil {
			changes["title"] = map[string]any{"from": strVal(beforeMap, "title"), "to": *req.Title}
		}
		if req.Artist != nil {
			changes["artist"] = map[string]any{"from": strVal(beforeMap, "artist"), "to": req.Artist}
		}
		if req.BaseChord != nil {
			changes["base_chord"] = map[string]any{"from": strVal(beforeMap, "base_chord"), "to": *req.BaseChord}
		}
		if req.LyricsAndChord != nil {
			changes["lyrics_and_chords"] = "updated"
		}
		if req.ExternalLinks != nil {
			changes["external_links"] = *req.ExternalLinks
		}
		entityName := strVal(beforeMap, "title")
		h.audit.Log(&uid, cl.Name, cl.Email, "update", "song", &id, entityName, changes)
	}
	return utils.OK(c, 200, "Song updated successfully", fiber.Map{"id": id})
}

func (h *Handler) DeleteSong(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid song ID")
	}
	before, _, _ := h.songs.GetByID(id)
	ok, err := h.songs.Delete(id)
	if err != nil {
		return utils.Fail(c, 500, "Failed to delete song")
	}
	if !ok {
		return utils.Fail(c, 404, "Song not found")
	}
	cl := middleware.GetClaims(c)
	if cl != nil {
		uid := cl.UserID
		entityName := strVal(before, "title")
		h.audit.Log(&uid, cl.Name, cl.Email, "delete", "song", &id, entityName, nil)
	}
	return utils.OK(c, 200, "Song deleted successfully", fiber.Map{"id": id})
}

func strVal(m map[string]any, key string) string {
if m == nil {
return ""
}
if v, ok := m[key]; ok {
if s, ok := v.(string); ok {
return s
}
}
return ""
}
