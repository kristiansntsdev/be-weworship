package services

import (
	"be-songbanks-v1/api/models"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

type PlaylistService struct {
	playlists *repositories.PlaylistRepository
	teams     *repositories.TeamRepository
	songs     *repositories.SongRepository
	clientURL string
}

func NewPlaylistService(p *repositories.PlaylistRepository, t *repositories.TeamRepository, s *repositories.SongRepository, clientURL string) *PlaylistService {
	return &PlaylistService{playlists: p, teams: t, songs: s, clientURL: clientURL}
}

func (s *PlaylistService) Create(userID int, name string, songs []int) (map[string]any, int, error) {
	if strings.TrimSpace(name) == "" {
		return nil, 400, fmt.Errorf("playlist_name is required")
	}
	exists, err := s.playlists.NameExistsForUser(userID, name)
	if err != nil {
		return nil, 500, err
	}
	if exists {
		return nil, 409, fmt.Errorf("a playlist with this name already exists")
	}
	id, err := s.playlists.Create(userID, name, songs)
	if err != nil {
		return nil, 500, err
	}
	return map[string]any{"id": id, "playlist_name": name, "user_id": userID, "songs": songs}, 201, nil
}

func (s *PlaylistService) List(userID, page, limit int) ([]map[string]any, map[string]any, error) {
	total, err := s.playlists.CountAccessible(userID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.playlists.ListAccessible(userID, page, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":             r.ID,
			"playlist_name":  r.PlaylistName,
			"user_id":        r.UserID,
			"songs":          utils.ParseIntSlice(r.SongsRaw.String),
			"playlist_notes": utils.ParseAnyJSON(r.NotesRaw.String),
			"createdAt":      utils.NullableTime(r.CreatedAt),
			"updatedAt":      utils.NullableTime(r.UpdatedAt),
			"access_type":    r.AccessType,
		})
	}
	return out, map[string]any{"currentPage": page, "totalPages": utils.Ceil(total, limit), "totalItems": total, "itemsPerPage": limit}, nil
}

func (s *PlaylistService) GetByIDWithAccess(playlistID, userID int) (map[string]any, int, error) {
	pl, access, err := s.loadWithAccess(playlistID, userID)
	if err == sql.ErrNoRows {
		return nil, 404, fmt.Errorf("playlist not found or access denied")
	}
	if err != nil {
		return nil, 500, err
	}
	resp := map[string]any{"id": pl.ID, "playlist_name": pl.PlaylistName, "user_id": pl.UserID, "songs": utils.ParseIntSlice(pl.SongsRaw.String), "playlist_notes": utils.ParseAnyJSON(pl.PlaylistNotesRaw.String), "createdAt": pl.CreatedAt, "updatedAt": pl.UpdatedAt, "access_type": access}
	if pl.IsShared {
		resp["sharable_link"] = utils.NullableString(pl.ShareableURL)
		resp["share_token"] = utils.NullableString(pl.ShareToken)
		resp["playlist_team_id"] = pl.PlaylistTeamID
		resp["is_shared"] = pl.IsShared
		resp["is_locked"] = pl.IsLocked
	}
	return resp, 200, nil
}

func (s *PlaylistService) UpdateName(playlistID, userID int, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		return 400, fmt.Errorf("playlist_name is required")
	}
	affected, err := s.playlists.UpdateName(playlistID, userID, name)
	if err != nil {
		return 500, err
	}
	if affected == 0 {
		return 404, fmt.Errorf("playlist not found")
	}
	return 200, nil
}

func (s *PlaylistService) Delete(playlistID, userID int) (int, error) {
	affected, err := s.playlists.Delete(playlistID, userID)
	if err != nil {
		return 500, err
	}
	if affected == 0 {
		return 404, fmt.Errorf("playlist not found")
	}
	return 200, nil
}

func (s *PlaylistService) GenerateSharelink(playlistID, userID int) (map[string]any, int, error) {
	ownerID, err := s.playlists.ExistsAndOwner(playlistID)
	if err != nil {
		return nil, 500, err
	}
	if ownerID == 0 {
		return nil, 404, fmt.Errorf("playlist not found")
	}
	if ownerID != userID {
		return nil, 403, fmt.Errorf("access denied")
	}

	token := fmt.Sprintf("%d-%d", playlistID, time.Now().UnixNano())
	link := strings.TrimRight(s.clientURL, "/") + "/playlist/join/" + token

	team, err := s.teams.FindByPlaylistID(playlistID)
	if err != nil {
		return nil, 500, err
	}
	var teamID int64
	if team == nil {
		teamID, err = s.teams.Create(playlistID, userID, []int{userID})
		if err != nil {
			return nil, 500, err
		}
	} else {
		teamID = int64(team.ID)
	}

	if err := s.playlists.UpdateShare(playlistID, link, token, teamID); err != nil {
		return nil, 500, err
	}

	return map[string]any{"playlist_id": playlistID, "playlist_team_id": teamID, "share_token": token, "sharable_link": link}, 201, nil
}

func (s *PlaylistService) Join(shareToken string, userID int) (map[string]any, int, error) {
	playlistID, ownerID, teamID, err := s.playlists.FindByShareToken(shareToken)
	if err == sql.ErrNoRows {
		return nil, 404, fmt.Errorf("invalid or expired share link")
	}
	if err != nil {
		return nil, 500, err
	}
	if ownerID == userID {
		return map[string]any{"playlist_id": playlistID}, 201, nil
	}

	if !teamID.Valid {
		newTeamID, err := s.teams.Create(playlistID, ownerID, []int{ownerID, userID})
		if err != nil {
			return nil, 500, err
		}
		if err := s.playlists.SetTeamID(playlistID, newTeamID); err != nil {
			return nil, 500, err
		}
		teamID = sql.NullInt64{Int64: newTeamID, Valid: true}
	} else {
		team, err := s.teams.GetByID(int(teamID.Int64))
		if err != nil || team == nil {
			return nil, 500, fmt.Errorf("failed to join playlist")
		}
		members := utils.ParseIntSlice(team.MembersRaw.String)
		if !utils.ContainsInt(members, userID) {
			members = append(members, userID)
			if err := s.teams.UpdateMembers(team.ID, members); err != nil {
				return nil, 500, err
			}
		}
	}

	return map[string]any{"playlist_id": playlistID, "playlist_team_id": teamID.Int64}, 201, nil
}

func (s *PlaylistService) AddSongs(playlistID, userID int, songIDs []int) (int, error) {
	if len(songIDs) == 0 {
		return 400, fmt.Errorf("songIds is required")
	}
	canManage, err := s.playlists.CanManage(playlistID, userID)
	if err != nil {
		return 500, err
	}
	if !canManage {
		return 403, fmt.Errorf("access denied")
	}
	if err := s.appendSongs(playlistID, songIDs); err != nil {
		return 500, err
	}
	return 200, nil
}

func (s *PlaylistService) AddSongWithBaseChord(playlistID, userID, songID int, baseChord string) (int, error) {
	if strings.TrimSpace(baseChord) == "" {
		return 400, fmt.Errorf("base_chord is required")
	}
	return s.AddSongs(playlistID, userID, []int{songID})
}

func (s *PlaylistService) RemoveSong(playlistID, userID, songID int) (int, error) {
	canManage, err := s.playlists.CanManage(playlistID, userID)
	if err != nil {
		return 500, err
	}
	if !canManage {
		return 403, fmt.Errorf("access denied")
	}
	pl, err := s.playlists.GetByID(playlistID)
	if err != nil || pl == nil {
		return 404, fmt.Errorf("playlist not found")
	}
	songs := utils.ParseIntSlice(pl.SongsRaw.String)
	if !utils.ContainsInt(songs, songID) {
		return 404, fmt.Errorf("song not found in playlist")
	}
	next := make([]int, 0, len(songs))
	for _, id := range songs {
		if id != songID {
			next = append(next, id)
		}
	}
	if err := s.playlists.SetSongs(playlistID, next); err != nil {
		return 500, err
	}
	return 200, nil
}

func (s *PlaylistService) appendSongs(playlistID int, songIDs []int) error {
	pl, err := s.playlists.GetByID(playlistID)
	if err != nil || pl == nil {
		return fmt.Errorf("playlist not found")
	}
	songs := utils.ParseIntSlice(pl.SongsRaw.String)
	set := map[int]struct{}{}
	for _, id := range songs {
		set[id] = struct{}{}
	}
	for _, id := range songIDs {
		exists, err := s.songs.ExistsByID(id)
		if err != nil || !exists {
			continue
		}
		if _, ok := set[id]; !ok {
			songs = append(songs, id)
			set[id] = struct{}{}
		}
	}
	return s.playlists.SetSongs(playlistID, songs)
}

func (s *PlaylistService) loadWithAccess(playlistID, userID int) (*models.Playlist, string, error) {
	pl, err := s.playlists.GetByID(playlistID)
	if err != nil || pl == nil {
		return nil, "", sql.ErrNoRows
	}
	if pl.UserID == userID {
		return pl, "owner", nil
	}
	team, err := s.teams.FindByPlaylistID(playlistID)
	if err != nil || team == nil {
		return nil, "", sql.ErrNoRows
	}
	if team.LeadID == userID {
		return pl, "leader", nil
	}
	if utils.ContainsInt(utils.ParseIntSlice(team.MembersRaw.String), userID) {
		return pl, "member", nil
	}
	return nil, "", sql.ErrNoRows
}
