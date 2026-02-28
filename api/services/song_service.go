package services

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"be-songbanks-v1/api/platform"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type SongService struct {
	songs *repositories.SongRepository
	tags  *repositories.TagRepository
	cache *platform.SongCache
}

func NewSongService(songRepo *repositories.SongRepository, tagRepo *repositories.TagRepository, cache *platform.SongCache) *SongService {
	return &SongService{songs: songRepo, tags: tagRepo, cache: cache}
}

func (s *SongService) Artists() ([]string, error) {
	raws, err := s.songs.ListArtistsRaw()
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, r := range raws {
		for _, a := range utils.ParseArtists(r) {
			name := strings.TrimSpace(a)
			if name != "" {
				set[name] = struct{}{}
			}
		}
	}
	artists := make([]string, 0, len(set))
	for v := range set {
		artists = append(artists, v)
	}
	return artists, nil
}

func (s *SongService) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int) ([]map[string]any, map[string]any, error) {
	cacheKey := buildSongsCacheKey(page, limit, search, baseChord, sortBy, sortOrder, tagIDs)
	if s.cache != nil && s.cache.Enabled() {
		var cached struct {
			Data       []map[string]any `json:"data"`
			Pagination map[string]any   `json:"pagination"`
		}
		if s.cache.Get(cacheKey, &cached) {
			log.Printf("[songs-cache] hit key=%s", cacheKey)
			return cached.Data, cached.Pagination, nil
		}
		log.Printf("[songs-cache] miss key=%s", cacheKey)
	} else {
		log.Printf("[songs-cache] disabled key=%s", cacheKey)
	}

	sortMap := map[string]string{"createdAt": "s.createdAt", "updatedAt": "s.updatedAt", "title": "s.title"}
	mappedSort := sortMap[sortBy]
	if mappedSort == "" {
		mappedSort = "s.createdAt"
	}
	if strings.ToUpper(sortOrder) != "ASC" {
		sortOrder = "DESC"
	}

	rows, total, err := s.songs.List(page, limit, search, baseChord, mappedSort, strings.ToUpper(sortOrder), tagIDs)
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
		data = append(data, map[string]any{
			"id":                r.ID,
			"title":             r.Title,
			"artist":            utils.ParseArtists(r.Artist.String),
			"base_chord":        utils.NullableString(r.BaseChord),
			"lyrics_and_chords": utils.NullableString(r.LyricsAndChord),
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

	if s.cache != nil && s.cache.Enabled() {
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
	return map[string]any{
		"id":                row.ID,
		"title":             row.Title,
		"artist":            utils.ParseArtists(row.Artist.String),
		"base_chord":        utils.NullableString(row.BaseChord),
		"lyrics_and_chords": utils.NullableString(row.LyricsAndChord),
		"createdAt":         utils.NullableTime(row.CreatedAt),
		"updatedAt":         utils.NullableTime(row.UpdatedAt),
		"tags":              tagRows,
	}, true, nil
}

func (s *SongService) Create(title string, artist any, baseChord, lyrics *string, tagNames []string) (map[string]any, error) {
	artistJSON := utils.MustArtistJSON(artist)
	songID, err := s.songs.Create(title, artistJSON, baseChord, lyrics)
	if err != nil {
		return nil, err
	}
	if err := s.assignTags(songID, tagNames); err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.InvalidateSongsList()
		log.Printf("[songs-cache] invalidate reason=create song_id=%d", songID)
	}
	return map[string]any{"id": songID, "title": title}, nil
}

func (s *SongService) Update(songID int, title *string, artist any, baseChord, lyrics *string, tagNames []string, tagNamesProvided bool) (bool, error) {
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
	if lyrics != nil {
		parts = append(parts, "lyrics_and_chords=?")
		args = append(args, lyrics)
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
		log.Printf("[songs-cache] invalidate reason=delete song_id=%d", songID)
	}
	return affected > 0, nil
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

func buildSongsCacheKey(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int) string {
	ids := append([]int(nil), tagIDs...)
	sort.Ints(ids)
	tagPart := ""
	for i, id := range ids {
		if i > 0 {
			tagPart += ","
		}
		tagPart += fmt.Sprintf("%d", id)
	}
	return fmt.Sprintf("songs:list:page=%d:limit=%d:search=%s:base=%s:sortBy=%s:sortOrder=%s:tags=%s",
		page, limit, strings.TrimSpace(search), strings.TrimSpace(baseChord), sortBy, strings.ToUpper(sortOrder), tagPart)
}
