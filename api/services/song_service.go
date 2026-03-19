package services

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/platform"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type SongService struct {
	songs     *repositories.SongRepository
	tags      *repositories.TagRepository
	playlists *repositories.PlaylistRepository
	cache     *platform.SongCache
}

func NewSongService(songRepo *repositories.SongRepository, tagRepo *repositories.TagRepository, playlistRepo *repositories.PlaylistRepository, cache *platform.SongCache) *SongService {
	return &SongService{songs: songRepo, tags: tagRepo, playlists: playlistRepo, cache: cache}
}

// parseExternalLinks parses a JSON external_links string into individual URL fields.
func parseExternalLinks(raw string) (spotify, youtube, appleMusic *string) {
	if raw == "" {
		return nil, nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, nil, nil
	}
	if v, ok := m["spotify"]; ok && v != "" {
		spotify = &v
	}
	if v, ok := m["youtube"]; ok && v != "" {
		youtube = &v
	}
	if v, ok := m["apple_music"]; ok && v != "" {
		appleMusic = &v
	}
	return
}

func (s *SongService) Artists() ([]map[string]any, error) {
	const cacheKey = "artists:list"
	if s.cache != nil && s.cache.Enabled() {
		var cached []map[string]any
		if s.cache.Get(cacheKey, &cached) {
			log.Printf("[artists-cache] hit")
			return cached, nil
		}
		log.Printf("[artists-cache] miss")
	} else {
		log.Printf("[artists-cache] disabled")
	}

	raws, err := s.songs.ListArtistsRaw()
	if err != nil {
		return nil, err
	}
	// Count songs per artist name
	counts := map[string]int{}
	for _, r := range raws {
		for _, a := range utils.ParseArtists(r) {
			name := strings.TrimSpace(a)
			if name != "" {
				counts[name]++
			}
		}
	}
	artists := make([]map[string]any, 0, len(counts))
	for name, count := range counts {
		artists = append(artists, map[string]any{
			"id":    utils.Slugify(name),
			"name":  name,
			"count": count,
		})
	}
	// Sort alphabetically by name
	sort.Slice(artists, func(i, j int) bool {
		return artists[i]["name"].(string) < artists[j]["name"].(string)
	})

	if s.cache != nil && s.cache.Enabled() {
		s.cache.Set(cacheKey, artists)
		log.Printf("[artists-cache] set")
	}
	return artists, nil
}

func (s *SongService) HomeStats() (map[string]any, error) {
	songCount, err := s.songs.Count()
	if err != nil {
		return nil, err
	}
	artists, err := s.Artists()
	if err != nil {
		return nil, err
	}
	shareableCount, err := s.playlists.CountShareable()
	if err != nil {
		return nil, err
	}
	weeklyShareCount, err := s.playlists.CountShareEventsThisWeek()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"song_count":               songCount,
		"artist_count":             len(artists),
		"shareable_playlist_count": shareableCount,
		"weekly_share_count":       weeklyShareCount,
	}, nil
}

func (s *SongService) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, hasLink, chordPro *bool, useCache bool) ([]map[string]any, map[string]any, error) {
	cacheKey := buildSongsCacheKey(page, limit, search, baseChord, sortBy, sortOrder, tagIDs, hasLink, chordPro)
	if useCache && s.cache != nil && s.cache.Enabled() {
		var cached struct {
			Data       []map[string]any `json:"data"`
			Pagination map[string]any   `json:"pagination"`
		}
		if s.cache.Get(cacheKey, &cached) {
			log.Printf("[songs-cache] hit key=%s", cacheKey)
			return cached.Data, cached.Pagination, nil
		}
		log.Printf("[songs-cache] miss key=%s", cacheKey)
	} else if !useCache {
		log.Printf("[songs-cache] bypassed (non-mobile) key=%s", cacheKey)
	} else {
		log.Printf("[songs-cache] disabled key=%s", cacheKey)
	}

	sortMap := map[string]string{"createdAt": `s."createdAt"`, "updatedAt": `s."updatedAt"`, "title": "s.title"}
	mappedSort := sortMap[sortBy]
	if mappedSort == "" {
		mappedSort = `s."createdAt"`
	}
	if strings.ToUpper(sortOrder) != "ASC" {
		sortOrder = "DESC"
	}

	rows, total, err := s.songs.List(page, limit, search, baseChord, mappedSort, strings.ToUpper(sortOrder), tagIDs, hasLink, chordPro)
	if err != nil {
		return nil, nil, err
	}

	data := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		tags, _ := s.tags.GetTagsForSong(r.ID)
		tagRows := make([]map[string]any, 0, len(tags))
		for _, t := range tags {
			tagRows = append(tagRows, map[string]any{"id": t.ID, "name": t.Name, "description": utils.NullableString(t.Description)})
		}
		sp, yt, am := parseExternalLinks(r.ExternalLinks.String)
		data = append(data, map[string]any{
			"id":         r.ID,
			"slug":       utils.NullableString(r.Slug),
			"title":      r.Title,
			"artist":     utils.ParseArtists(r.Artist.String),
			"base_chord": utils.NullableString(r.BaseChord),
			"bpm": func() any {
				if r.Bpm.Valid {
					return r.Bpm.Int64
				}
				return nil
			}(),
			"lyrics_and_chords": utils.NullableString(r.LyricsAndChord),
			"external_links":    utils.NullableString(r.ExternalLinks),
			"spotify_url":       sp,
			"youtube_url":       yt,
			"apple_music_url":   am,
			"dmca_takedown":     r.DmcaTakedown,
			"dmca_status_notes": utils.NullableString(r.DmcaStatusNotes),
			"createdAt":         utils.NullableTime(r.CreatedAt),
			"updatedAt":         utils.NullableTime(r.UpdatedAt),
			"tags":              tagRows,
		})
	}

	pagination := map[string]any{
		"currentPage":  page,
		"totalPages":   utils.Ceil(total, limit),
		"totalItems":   total,
		"itemsPerPage": limit,
		"hasNextPage":  page < utils.Ceil(total, limit),
		"hasPrevPage":  page > 1,
	}

	if useCache && s.cache != nil && s.cache.Enabled() {
		s.cache.Set(cacheKey, map[string]any{"data": data, "pagination": pagination})
		log.Printf("[songs-cache] set key=%s", cacheKey)
	}
	return data, pagination, nil
}

func (s *SongService) GetByID(id int) (map[string]any, bool, error) {
	row, err := s.songs.GetByID(id)
	if err != nil {
		return nil, false, err
	}
	if row == nil {
		return nil, false, nil
	}
	tags, _ := s.tags.GetTagsForSong(row.ID)
	tagRows := make([]map[string]any, 0, len(tags))
	for _, t := range tags {
		tagRows = append(tagRows, map[string]any{"id": t.ID, "name": t.Name, "description": utils.NullableString(t.Description)})
	}
	sp, yt, am := parseExternalLinks(row.ExternalLinks.String)
	return map[string]any{
		"id":         row.ID,
		"slug":       utils.NullableString(row.Slug),
		"title":      row.Title,
		"artist":     utils.ParseArtists(row.Artist.String),
		"base_chord": utils.NullableString(row.BaseChord),
		"bpm": func() any {
			if row.Bpm.Valid {
				return row.Bpm.Int64
			}
			return nil
		}(),
		"lyrics_and_chords": utils.NullableString(row.LyricsAndChord),
		"external_links":    utils.NullableString(row.ExternalLinks),
		"spotify_url":       sp,
		"youtube_url":       yt,
		"apple_music_url":   am,
		"dmca_takedown":     row.DmcaTakedown,
		"dmca_status_notes": utils.NullableString(row.DmcaStatusNotes),
		"createdAt":         utils.NullableTime(row.CreatedAt),
		"updatedAt":         utils.NullableTime(row.UpdatedAt),
		"tags":              tagRows,
	}, true, nil
}

func (s *SongService) GetBySlug(slug string) (map[string]any, bool, error) {
	row, err := s.songs.GetBySlug(slug)
	if err != nil {
		return nil, false, err
	}
	if row == nil {
		return nil, false, nil
	}
	tags, _ := s.tags.GetTagsForSong(row.ID)
	tagRows := make([]map[string]any, 0, len(tags))
	for _, t := range tags {
		tagRows = append(tagRows, map[string]any{"id": t.ID, "name": t.Name, "description": utils.NullableString(t.Description)})
	}
	sp2, yt2, am2 := parseExternalLinks(row.ExternalLinks.String)
	return map[string]any{
		"id":         row.ID,
		"slug":       utils.NullableString(row.Slug),
		"title":      row.Title,
		"artist":     utils.ParseArtists(row.Artist.String),
		"base_chord": utils.NullableString(row.BaseChord),
		"bpm": func() any {
			if row.Bpm.Valid {
				return row.Bpm.Int64
			}
			return nil
		}(),
		"lyrics_and_chords": utils.NullableString(row.LyricsAndChord),
		"external_links":    utils.NullableString(row.ExternalLinks),
		"spotify_url":       sp2,
		"youtube_url":       yt2,
		"apple_music_url":   am2,
		"dmca_takedown":     row.DmcaTakedown,
		"dmca_status_notes": utils.NullableString(row.DmcaStatusNotes),
		"createdAt":         utils.NullableTime(row.CreatedAt),
		"updatedAt":         utils.NullableTime(row.UpdatedAt),
		"tags":              tagRows,
	}, true, nil
}

func (s *SongService) Create(title string, artist any, baseChord *string, bpm *int, lyrics *string, externalLinks *string, dmcaTakedown bool, dmcaStatusNotes *string, tagNames []string) (map[string]any, error) {
	artistJSON := utils.MustArtistJSON(artist)
	songID, err := s.songs.Create(title, artistJSON, baseChord, bpm, lyrics, externalLinks, dmcaTakedown, dmcaStatusNotes, utils.Slugify(title))
	if err != nil {
		return nil, err
	}
	if err := s.assignTags(songID, tagNames); err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.InvalidateSongsList()
		s.cache.InvalidateArtists()
		log.Printf("[songs-cache] invalidate reason=create song_id=%d", songID)
	}
	return map[string]any{"id": songID, "title": title}, nil
}

func (s *SongService) Update(songID int, title *string, artist any, baseChord *string, bpm *int, lyrics *string, externalLinks *string, dmcaTakedown *bool, dmcaStatusNotes *string, tagNames []string, tagNamesProvided bool) (bool, error) {
	parts := []string{}
	args := []any{}
	if title != nil {
		parts = append(parts, "title=?")
		args = append(args, *title)
	}
	if artist != nil {
		parts = append(parts, "artist=?")
		args = append(args, utils.MustArtistJSON(artist))
	}
	if baseChord != nil {
		parts = append(parts, "base_chord=?")
		args = append(args, baseChord)
	}
	if bpm != nil {
		parts = append(parts, "bpm=?")
		args = append(args, *bpm)
	}
	if lyrics != nil {
		parts = append(parts, "lyrics_and_chords=?")
		args = append(args, lyrics)
	}
	if externalLinks != nil {
		parts = append(parts, "external_links=?")
		args = append(args, externalLinks)
	}
	if dmcaTakedown != nil {
		parts = append(parts, "dmca_takedown=?")
		args = append(args, *dmcaTakedown)
	}
	if dmcaStatusNotes != nil {
		parts = append(parts, "dmca_status_notes=?")
		args = append(args, dmcaStatusNotes)
	}

	if len(parts) > 0 {
		affected, err := s.songs.UpdateFields(songID, strings.Join(parts, ","), args...)
		if err != nil {
			return false, err
		}
		if affected == 0 {
			return false, nil
		}
	}

	if tagNamesProvided {
		if err := s.songs.ClearSongTags(songID); err != nil {
			return false, err
		}
		if err := s.assignTags(songID, tagNames); err != nil {
			return false, err
		}
	}

	if s.cache != nil {
		s.cache.InvalidateSongsList()
		s.cache.InvalidateArtists()
		log.Printf("[songs-cache] invalidate reason=update song_id=%d", songID)
	}
	return true, nil
}

func (s *SongService) Delete(songID int) (bool, error) {
	affected, err := s.songs.DeleteByID(songID)
	if err != nil {
		return false, err
	}
	if affected > 0 && s.cache != nil {
		s.cache.InvalidateSongsList()
		s.cache.InvalidateArtists()
		log.Printf("[songs-cache] invalidate reason=delete song_id=%d", songID)
	}
	return affected > 0, nil
}

func (s *SongService) ExportSongs() ([]map[string]any, error) {
	rows, err := s.songs.ListAllChordPro()
	if err != nil {
		return nil, err
	}
	data := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		sp, yt, am := parseExternalLinks(r.ExternalLinks.String)
		data = append(data, map[string]any{
			"id":                r.ID,
			"title":             r.Title,
			"artist":            utils.ParseArtists(r.Artist.String),
			"base_chord":        utils.NullableString(r.BaseChord),
			"bpm": func() any {
				if r.Bpm.Valid {
					return r.Bpm.Int64
				}
				return nil
			}(),
			"lyrics_and_chords": utils.NullableString(r.LyricsAndChord),
			"spotify_url":       sp,
			"youtube_url":       yt,
			"apple_music_url":   am,
		})
	}
	return data, nil
}

func (s *SongService) assignTags(songID int, names []string) error {
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		t, err := s.tags.FindByName(name)
		if err != nil {
			return err
		}
		var tagID int
		if t == nil {
			tagID, err = s.tags.Create(name)
			if err != nil {
				return err
			}
		} else {
			tagID = t.ID
		}
		if err := s.songs.AssignSongTag(songID, tagID); err != nil {
			return fmt.Errorf("assign tag: %w", err)
		}
	}
	return nil
}

func buildSongsCacheKey(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, hasLink, chordPro *bool) string {
	ids := append([]int(nil), tagIDs...)
	sort.Ints(ids)
	tagPart := ""
	for i, id := range ids {
		if i > 0 {
			tagPart += ","
		}
		tagPart += fmt.Sprintf("%d", id)
	}
	boolPart := func(v *bool) string {
		if v == nil {
			return "any"
		}
		if *v {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("songs:list:page=%d:limit=%d:search=%s:base=%s:sortBy=%s:sortOrder=%s:tags=%s:hasLink=%s:chordPro=%s",
		page, limit, strings.TrimSpace(search), strings.TrimSpace(baseChord), sortBy, strings.ToUpper(sortOrder), tagPart, boolPart(hasLink), boolPart(chordPro))
}

// ── Song Requests ─────────────────────────────────────────────────────────────

func (s *SongService) RequestSong(userID int, songTitle, referenceLink string) (*models.SongRequest, error) {
	if strings.TrimSpace(songTitle) == "" {
		return nil, fmt.Errorf("song_title is required")
	}
	if strings.TrimSpace(referenceLink) == "" {
		return nil, fmt.Errorf("reference_link is required")
	}
	return s.songs.CreateSongRequest(userID, strings.TrimSpace(songTitle), strings.TrimSpace(referenceLink))
}

func (s *SongService) ListSongRequests(status string, page, limit int) ([]models.SongRequest, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.songs.ListSongRequests(status, page, limit)
}

func (s *SongService) UpdateSongRequestStatus(id int, status, adminNotes string) error {
	validStatuses := map[string]bool{"pending": true, "approved": true, "rejected": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: must be pending, approved, or rejected")
	}
	return s.songs.UpdateSongRequestStatus(id, status, adminNotes)
}
