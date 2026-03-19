package handlers

import (
	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/utils"
	"bytes"
	"log"

	"github.com/gofiber/fiber/v2"
	"strings"
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
		t := true
		hasLink = &t
	} else if v == "false" {
		f := false
		hasLink = &f
	}
	var chordPro *bool
	if v := c.Query("chordpro"); v == "true" {
		t := true
		chordPro = &t
	} else if v == "false" {
		f := false
		chordPro = &f
	}
	isMobileClient := isMobileUserAgent(c.Get("User-Agent"))
	// Mobile app should only receive songs already in ChordPro format.
	if chordPro == nil && isMobileClient {
		t := true
		chordPro = &t
	}
	data, pagination, err := h.songs.List(page, limit, c.Query("search"), c.Query("base_chord"), c.Query("sortBy", "createdAt"), c.Query("sortOrder", "DESC"), utils.ParseCSVInts(c.Query("tag_ids")), hasLink, chordPro, isMobileClient)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve songs")
	}
	return c.JSON(fiber.Map{"code": 200, "message": "Songs retrieved successfully", "data": data, "pagination": pagination})
}

func isMobileUserAgent(ua string) bool {
	ua = strings.ToLower(ua)
	return strings.Contains(ua, "okhttp") ||
		strings.Contains(ua, "cfnetwork") ||
		strings.Contains(ua, "expo") ||
		strings.Contains(ua, "reactnative")
}

func (h *Handler) GetSongsExport(c *fiber.Ctx) error {
	data, err := h.songs.ExportSongs()
	if err != nil {
		return utils.Fail(c, 500, "Failed to export songs")
	}
	return utils.OK(c, 200, "Songs exported successfully", data)
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
	// Notify all subscribers that a new song is available.
	// If LyricsAndChord is set the song is already ChordPro-ready; notify immediately.
	// Songs without lyrics will trigger a notification when lyrics are added via UpdateSong.
	if req.LyricsAndChord != nil && *req.LyricsAndChord != "" {
		h.notifications.NotifyNewSong(req.Title)
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
	// Fire notification when lyrics_and_chords is being added for the first time
	// (song transitions to ChordPro-ready status).
	hasNewChords := req.LyricsAndChord != nil && *req.LyricsAndChord != ""
	hadChords := strVal(beforeMap, "lyrics_and_chords") != ""
	log.Printf("[song] UpdateSong id=%d: hasNewChords=%v hadChords=%v → notify=%v", id, hasNewChords, hadChords, hasNewChords && !hadChords)
	if hasNewChords && !hadChords {
		title := strVal(beforeMap, "title")
		if req.Title != nil && *req.Title != "" {
			title = *req.Title
		}
		h.notifications.NotifyNewSong(title)
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

// ── Song Requests ─────────────────────────────────────────────────────────────

func (h *Handler) RequestSong(c *fiber.Ctx) error {
	var req struct {
		SongTitle     string `json:"song_title"`
		ReferenceLink string `json:"reference_link"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	result, err := h.songs.RequestSong(cl.UserID, req.SongTitle, req.ReferenceLink)
	if err != nil {
		if err.Error() == "song_title is required" || err.Error() == "reference_link is required" {
			return utils.Fail(c, 400, err.Error())
		}
		return utils.Fail(c, 500, "Failed to submit song request")
	}
	return utils.OK(c, 201, "Song request submitted successfully", result)
}

func (h *Handler) GetSongRequests(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	status := c.Query("status")
	data, total, err := h.songs.ListSongRequests(status, page, limit)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve song requests")
	}
	return utils.OK(c, 200, "Song requests retrieved successfully", fiber.Map{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) UpdateSongRequest(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid request ID")
	}
	var req struct {
		Status     string `json:"status"`
		AdminNotes string `json:"admin_notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "Invalid JSON")
	}
	// Fetch request before update so we have the title and requester ID
	songReq, found, _ := h.songs.GetSongRequestByID(id)
	if err := h.songs.UpdateSongRequestStatus(id, req.Status, req.AdminNotes); err != nil {
		if err.Error() == "invalid status: must be pending, approved, or rejected" {
			return utils.Fail(c, 400, err.Error())
		}
		return utils.Fail(c, 500, "Failed to update song request")
	}
	// Notify the requester when status changes to approved or rejected
	if found && songReq != nil && (req.Status == "approved" || req.Status == "rejected") {
		h.notifications.NotifySongRequestUpdated(songReq.SongTitle, req.Status, songReq.UserID)
	}
	return utils.OK(c, 200, "Song request updated successfully", nil)
}

func (h *Handler) GetMySongRequests(c *fiber.Ctx) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	data, total, err := h.songs.ListMySongRequests(cl.UserID, page, limit)
	if err != nil {
		return utils.Fail(c, 500, "Failed to retrieve song requests")
	}
	return utils.OK(c, 200, "Song requests retrieved successfully", fiber.Map{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) DeleteSongRequest(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return utils.Fail(c, 400, "Invalid request ID")
	}
	cl := middleware.GetClaims(c)
	if cl == nil {
		return utils.Fail(c, 401, "Unauthorized")
	}
	if err := h.songs.DeleteSongRequest(id, cl.UserID); err != nil {
		switch err.Error() {
		case "not found":
			return utils.Fail(c, 404, "Song request not found")
		case "forbidden":
			return utils.Fail(c, 403, "You can only delete your own song requests")
		default:
			return utils.Fail(c, 500, "Failed to delete song request")
		}
	}
	return utils.OK(c, 200, "Song request deleted successfully", nil)
}
