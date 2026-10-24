package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/platform"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type SongService struct {
	songs      repositories.SongRepoIface
	tags       repositories.TagRepoIface
	playlists  repositories.PlaylistRepoIface
	favorites  repositories.FavoriteRepoIface // nil-safe
	cache      platform.SongCacheIface
	embeddings *SongEmbeddingService // nil = semantic search disabled
}

// embeddings may be nil (no embedding service configured): search then ranks by text only.
// favorites may be nil (favorites disabled).
func NewSongService(songRepo *repositories.SongRepository, tagRepo *repositories.TagRepository, playlistRepo *repositories.PlaylistRepository, cache *platform.SongCache, embeddings *SongEmbeddingService, favorites repositories.FavoriteRepoIface) *SongService {
	return &SongService{songs: songRepo, tags: tagRepo, playlists: playlistRepo, cache: cache, embeddings: embeddings, favorites: favorites}
}

const (
	semanticSearchTimeout = 2500 * time.Millisecond
	songIndexTimeout      = 20 * time.Second
	rrfK                  = 60.0 // reciprocal rank fusion constant
)

// parseExternalLinks parses a JSON external_links string into individual URL fields.
// The admin saves keys as "Spotify", "Youtube", "Apple Music" (display names).
func parseExternalLinks(raw string) (spotify, youtube, appleMusic *string) {
	if raw == "" {
		return nil, nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, nil, nil
	}
	get := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := m[k]; ok && v != "" {
				return &v
			}
		}
		return nil
	}
	spotify = get("Spotify", "spotify")
	youtube = get("Youtube", "YouTube", "youtube")
	appleMusic = get("Apple Music", "apple_music", "apple music", "AppleMusic")
	return
}

func (s *SongService) Artists() ([]map[string]any, error) {
	raws, err := s.songs.ListArtistsRaw()
	if err != nil {
		return nil, err
	}
	// Count ChordPro songs per artist name
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
	return artists, nil
}

// Keys returns the distinct song keys (base_chord) with song counts, for key filters.
func (s *SongService) Keys() ([]repositories.KeyCount, error) {
	return s.songs.ListKeyCounts()
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

func (s *SongService) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, keys, artists []string, hasLink, chordPro *bool, useCache bool) ([]map[string]any, map[string]any, error) {
	cacheKey := buildSongsCacheKey(page, limit, search, baseChord, sortBy, sortOrder, tagIDs, keys, artists, hasLink, chordPro)
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

	var rows []models.Song
	var total int
	var err error
	if sortBy == "relevance" && strings.TrimSpace(search) != "" {
		rows, total, err = s.rankedSearch(page, limit, search, baseChord, tagIDs, keys, artists, hasLink, chordPro)
	} else {
		sortMap := map[string]string{"createdAt": `s."createdAt"`, "updatedAt": `s."updatedAt"`, "title": "s.title", "bpm": "s.bpm", "base_chord": "s.base_chord"}
		mappedSort := sortMap[sortBy]
		if sortBy == "relevance" {
			// Nothing to rank without a query: fall back to A–Z.
			mappedSort, sortOrder = "s.title", "ASC"
		}
		if mappedSort == "" {
			mappedSort = `s."createdAt"`
		}
		if strings.ToUpper(sortOrder) != "ASC" {
			sortOrder = "DESC"
		}
		rows, total, err = s.songs.List(page, limit, search, baseChord, mappedSort, strings.ToUpper(sortOrder), tagIDs, keys, artists, hasLink, chordPro)
	}
	if err != nil {
		return nil, nil, err
	}

	songIDs := make([]int, len(rows))
	for i, r := range rows {
		songIDs[i] = r.ID
	}
	tagsBySong, _ := s.tags.GetTagsForSongs(songIDs)

	data := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		tags := tagsBySong[r.ID]
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

// rankedSearch orders search results by relevance. Text matches (exact title, phrase, all words, typo-tolerant
// trigram) and semantic matches (bge-m3 embeddings of the title/lyrics/stanzas) are ranked separately and merged
// with reciprocal rank fusion. Without an embedder, or when it errors or times out, it ranks by text alone.
func (s *SongService) rankedSearch(page, limit int, search, baseChord string, tagIDs []int, keys, artists []string, hasLink, chordPro *bool) ([]models.Song, int, error) {
	var semanticIDs []int
	if s.embeddings != nil {
		ctx, cancel := context.WithTimeout(context.Background(), semanticSearchTimeout)
		matches, err := s.embeddings.SemanticMatches(ctx, search)
		cancel()
		if err != nil {
			log.Printf("[search] semantic search unavailable, ranking by text only: %v", err)
		}
		for _, m := range matches {
			semanticIDs = append(semanticIDs, m.SongID)
		}
	}

	cands, err := s.songs.SearchCandidates(search, semanticIDs, baseChord, tagIDs, keys, artists, hasLink, chordPro)
	if err != nil {
		return nil, 0, err
	}
	ordered := rankCandidates(cands, semanticIDs)

	total := len(ordered)
	start := min((page-1)*limit, total)
	pageIDs := ordered[start:min(start+limit, total)]
	rows, err := s.songs.ListByIDs(pageIDs)
	if err != nil {
		return nil, 0, err
	}
	byID := make(map[int]models.Song, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	out := make([]models.Song, 0, len(pageIDs))
	for _, id := range pageIDs {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, total, nil
}

// rankCandidates merges the text ranking (candidates with LexMatch, by LexScore) and the semantic ranking
// (semanticIDs, nearest first) with reciprocal rank fusion and returns candidate ids best first.
// Strong text matches always come first (exact title, then a verbatim line): someone typing a title or a line they
// know should not see a loosely related song above it. Semantic ids removed by filters are ignored.
func rankCandidates(cands []repositories.SearchCandidate, semanticIDs []int) []int {
	inCands := make(map[int]bool, len(cands))
	lex := make([]repositories.SearchCandidate, 0, len(cands))
	for _, c := range cands {
		inCands[c.ID] = true
		if c.LexMatch {
			lex = append(lex, c)
		}
	}
	sort.SliceStable(lex, func(i, j int) bool {
		if lex[i].LexScore != lex[j].LexScore {
			return lex[i].LexScore > lex[j].LexScore
		}
		return lex[i].Title < lex[j].Title
	})

	score := make(map[int]float64, len(cands))
	for rank, c := range lex {
		score[c.ID] += 1 / (rrfK + float64(rank+1))
	}
	rank := 0
	for _, id := range semanticIDs {
		if !inCands[id] {
			continue
		}
		rank++
		score[id] += 1 / (rrfK + float64(rank))
	}

	ordered := append([]repositories.SearchCandidate(nil), cands...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Strength != ordered[j].Strength {
			return ordered[i].Strength > ordered[j].Strength
		}
		si, sj := score[ordered[i].ID], score[ordered[j].ID]
		if si != sj {
			return si > sj
		}
		if ordered[i].LexScore != ordered[j].LexScore {
			return ordered[i].LexScore > ordered[j].LexScore
		}
		if ordered[i].Title != ordered[j].Title {
			return ordered[i].Title < ordered[j].Title
		}
		return ordered[i].ID < ordered[j].ID
	})
	ids := make([]int, len(ordered))
	for i, c := range ordered {
		ids[i] = c.ID
	}
	return ids
}

// reindexSong refreshes a song's embeddings after an admin edit. It runs inline because serverless instances
// can be frozen after the response; a failure is only logged — the backfill retries it via the content hash.
func (s *SongService) reindexSong(songID int) {
	if s.embeddings == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), songIndexTimeout)
	defer cancel()
	if _, err := s.embeddings.IndexSong(ctx, songID); err != nil {
		log.Printf("[embeddings] song_id=%d not re-indexed (run the backfill to retry): %v", songID, err)
	}
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
	s.reindexSong(songID)
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
		// Auto-extract plain lyrics when lyrics are updated
		plainLyrics := utils.ExtractPlainLyrics(*lyrics)
		parts = append(parts, "plain_lyrics=?")
		args = append(args, plainLyrics)
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
	if title != nil || artist != nil || lyrics != nil {
		s.reindexSong(songID)
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
			"id":         r.ID,
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

func buildSongsCacheKey(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, keys, artists []string, hasLink, chordPro *bool) string {
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
	listPart := func(v []string) string {
		sorted := append([]string(nil), v...)
		sort.Strings(sorted)
		return url.QueryEscape(strings.Join(sorted, "|"))
	}
	return fmt.Sprintf("songs:list:page=%d:limit=%d:search=%s:base=%s:sortBy=%s:sortOrder=%s:tags=%s:keys=%s:artists=%s:hasLink=%s:chordPro=%s",
		page, limit, strings.TrimSpace(search), strings.TrimSpace(baseChord), sortBy, strings.ToUpper(sortOrder), tagPart, listPart(keys), listPart(artists), boolPart(hasLink), boolPart(chordPro))
}

// ── Song Requests ─────────────────────────────────────────────────────────────

func (s *SongService) RequestSong(userID int, songTitle, referenceLink, lyricsType, lyrics string) (*models.SongRequest, error) {
	if strings.TrimSpace(songTitle) == "" {
		return nil, fmt.Errorf("song_title is required")
	}
	referenceLink = strings.TrimSpace(referenceLink)
	lyrics = strings.TrimSpace(lyrics)
	if referenceLink == "" && lyrics == "" {
		return nil, fmt.Errorf("lyrics or reference_link is required")
	}
	lyricsType = strings.TrimSpace(lyricsType)
	if lyricsType == "" {
		lyricsType = "lyrics"
	}
	if lyricsType != "lyrics" && lyricsType != "lyrics_chords" {
		return nil, fmt.Errorf("invalid lyrics_type: must be lyrics or lyrics_chords")
	}
	return s.songs.CreateSongRequest(userID, strings.TrimSpace(songTitle), referenceLink, lyricsType, lyrics)
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
	validStatuses := map[string]bool{"pending": true, "in_progress": true, "approved": true, "rejected": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: must be pending, in_progress, approved, or rejected")
	}
	if err := s.songs.UpdateSongRequestStatus(id, status, adminNotes); err != nil {
		return err
	}
	if status == "approved" {
		return s.songs.ClearSongRequestLyrics(id)
	}
	return nil
}

func (s *SongService) ListMySongRequests(userID, page, limit int) ([]models.SongRequest, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.songs.ListUserSongRequests(userID, page, limit)
}

func (s *SongService) DeleteSongRequest(id, userID int) error {
	req, found, err := s.songs.GetSongRequestByID(id)
	if err != nil {
		return fmt.Errorf("internal error")
	}
	if !found {
		return fmt.Errorf("not found")
	}
	if req.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	_, err = s.songs.DeleteSongRequest(id, userID)
	return err
}

// GetSongRequestByID returns a song request by its ID.
func (s *SongService) GetSongRequestByID(id int) (*models.SongRequest, bool, error) {
	return s.songs.GetSongRequestByID(id)
}

// ── Song Reports ──────────────────────────────────────────────────────────────

func (s *SongService) CreateSongReport(userID, songID int, reportType, description, evidenceURL string) (*models.SongReport, error) {
	if songID < 1 {
		return nil, fmt.Errorf("song_id is required")
	}
	exists, err := s.songs.ExistsByID(songID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("song not found")
	}

	reportType = strings.TrimSpace(reportType)
	validTypes := map[string]bool{"lyrics": true, "chord": true, "other": true}
	if !validTypes[reportType] {
		return nil, fmt.Errorf("invalid report_type: must be lyrics, chord, or other")
	}

	description = strings.TrimSpace(description)
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}

	evidenceURL = strings.TrimSpace(evidenceURL)
	if !isHTTPURL(evidenceURL) {
		return nil, fmt.Errorf("evidence_url must be a valid http or https URL")
	}

	return s.songs.CreateSongReport(userID, songID, reportType, description, evidenceURL)
}

func isHTTPURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func (s *SongService) ListSongReports(status string, page, limit int) ([]models.SongReport, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.songs.ListSongReports(status, page, limit)
}

func (s *SongService) UpdateSongReportStatus(id int, status, adminNotes string) error {
	validStatuses := map[string]bool{"pending": true, "approved": true, "rejected": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: must be pending, approved, or rejected")
	}
	return s.songs.UpdateSongReportStatus(id, status, adminNotes)
}

func (s *SongService) ListMySongReports(userID, page, limit int) ([]models.SongReport, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.songs.ListUserSongReports(userID, page, limit)
}

func (s *SongService) DeleteSongReport(id, userID int) error {
	report, found, err := s.songs.GetSongReportByID(id)
	if err != nil {
		return fmt.Errorf("internal error")
	}
	if !found {
		return fmt.Errorf("not found")
	}
	if report.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	_, err = s.songs.DeleteSongReport(id, userID)
	return err
}

func (s *SongService) GetSongReportByID(id int) (*models.SongReport, bool, error) {
	return s.songs.GetSongReportByID(id)
}

// ── Favorites ────────────────────────────────────────────────────────────────

func (s *SongService) IsFavorite(userID, songID int) (bool, error) {
	if s.favorites == nil {
		return false, nil
	}
	return s.favorites.IsFavorite(userID, songID)
}

func (s *SongService) AddFavorite(userID, songID int) error {
	if s.favorites == nil {
		return fmt.Errorf("favorites unavailable")
	}
	exists, err := s.songs.ExistsByID(songID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("song not found")
	}
	return s.favorites.Add(userID, songID)
}

func (s *SongService) RemoveFavorite(userID, songID int) error {
	if s.favorites == nil {
		return fmt.Errorf("favorites unavailable")
	}
	return s.favorites.Remove(userID, songID)
}

// ToggleFavorite returns the new favorited state.
func (s *SongService) ToggleFavorite(userID, songID int) (bool, error) {
	if s.favorites == nil {
		return false, fmt.Errorf("favorites unavailable")
	}
	exists, err := s.songs.ExistsByID(songID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("song not found")
	}
	ok, err := s.favorites.IsFavorite(userID, songID)
	if err != nil {
		return false, err
	}
	if ok {
		return false, s.favorites.Remove(userID, songID)
	}
	if err := s.favorites.Add(userID, songID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SongService) ListFavorites(userID int) ([]map[string]any, error) {
	if s.favorites == nil {
		return []map[string]any{}, nil
	}
	ids, err := s.favorites.ListSongIDs(userID)
	if err != nil {
		return nil, err
	}
	return s.mapsForOrderedIDs(ids, true)
}

// ListByOrderedIDs maps song ids to list cards, preserving order (skips missing).
func (s *SongService) ListByOrderedIDs(ids []int) ([]map[string]any, error) {
	return s.mapsForOrderedIDs(ids, false)
}

func (s *SongService) mapsForOrderedIDs(ids []int, markFavorite bool) ([]map[string]any, error) {
	if len(ids) == 0 {
		return []map[string]any{}, nil
	}
	rows, err := s.songs.ListByIDs(ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]models.Song, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row, ok := byID[id]
		if !ok {
			continue
		}
		tags, _ := s.tags.GetTagsForSong(row.ID)
		tagRows := make([]map[string]any, 0, len(tags))
		for _, t := range tags {
			tagRows = append(tagRows, map[string]any{"id": t.ID, "name": t.Name, "description": utils.NullableString(t.Description)})
		}
		sp, yt, am := parseExternalLinks(row.ExternalLinks.String)
		item := map[string]any{
			"id":                row.ID,
			"slug":              utils.NullableString(row.Slug),
			"title":             row.Title,
			"artist":            utils.ParseArtists(row.Artist.String),
			"base_chord":        utils.NullableString(row.BaseChord),
			"lyrics_and_chords": utils.NullableString(row.LyricsAndChord),
			"spotify_url":       sp,
			"youtube_url":       yt,
			"apple_music_url":   am,
			"tags":              tagRows,
		}
		if markFavorite {
			item["is_favorite"] = true
		}
		out = append(out, item)
	}
	return out, nil
}
