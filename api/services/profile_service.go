package services

import (
	"errors"
	"strings"
	"time"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

const (
	profileSearchMinLength = 2
	profileSearchLimit     = 20
)

var ErrProfileNotFound = errors.New("user not found")

// ProfileService serves what signed-in users may see of each other: username, display name, avatar, verified
// badge and public playlists. Never email or address details.
type ProfileService struct {
	profiles repositories.ProfileRepoIface
}

func NewProfileService(profiles repositories.ProfileRepoIface) *ProfileService {
	return &ProfileService{profiles: profiles}
}

// ProfileSummary is one row in people search results.
type ProfileSummary struct {
	Username        string  `json:"username"`
	DisplayName     string  `json:"display_name"`
	AvatarURL       *string `json:"avatar_url"`
	Verified        bool    `json:"verified"`
	PublicPlaylists int     `json:"public_playlists"`
}

// PublicPlaylist is a public playlist as listed on its owner's profile.
type PublicPlaylist struct {
	ID           int       `json:"id"`
	PlaylistName string    `json:"playlist_name"`
	SongCount    int       `json:"song_count"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Profile is a user's public page.
type Profile struct {
	Username    string           `json:"username"`
	DisplayName string           `json:"display_name"`
	AvatarURL   *string          `json:"avatar_url"`
	Verified    bool             `json:"verified"`
	Playlists   []PublicPlaylist `json:"playlists"`
}

// Search finds people by username or display name. Queries shorter than two characters return nothing.
func (s *ProfileService) Search(q string) ([]ProfileSummary, error) {
	q = strings.Join(strings.Fields(strings.TrimPrefix(strings.TrimSpace(q), "@")), " ")
	out := []ProfileSummary{}
	if len([]rune(q)) < profileSearchMinLength {
		return out, nil
	}
	rows, err := s.profiles.Search(q, profileSearchLimit)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out = append(out, ProfileSummary{
			Username:        r.Username,
			DisplayName:     r.DisplayName,
			AvatarURL:       nullableStringPtr(r.AvatarURL.String, r.AvatarURL.Valid),
			Verified:        r.Verified,
			PublicPlaylists: r.PublicPlaylists,
		})
	}
	return out, nil
}

// Get returns an active user's profile and their public playlists, or ErrProfileNotFound.
func (s *ProfileService) Get(username string) (*Profile, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" {
		return nil, ErrProfileNotFound
	}
	row, err := s.profiles.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrProfileNotFound
	}
	playlists, err := s.profiles.PublicPlaylists(row.ID)
	if err != nil {
		return nil, err
	}
	out := make([]PublicPlaylist, 0, len(playlists))
	for _, p := range playlists {
		out = append(out, PublicPlaylist{
			ID:           p.ID,
			PlaylistName: p.PlaylistName,
			SongCount:    len(utils.ParseIntSlice(p.SongsRaw.String)),
			UpdatedAt:    p.UpdatedAt,
		})
	}
	return &Profile{
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   nullableStringPtr(row.AvatarURL.String, row.AvatarURL.Valid),
		Verified:    row.Verified,
		Playlists:   out,
	}, nil
}

func nullableStringPtr(s string, valid bool) *string {
	if !valid || strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
