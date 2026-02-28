package handlers

import (
	"fmt"
	"strconv"

	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/services"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	authMW    *middleware.AuthMiddleware
	auth      *services.AuthService
	songs     *services.SongService
	tags      *services.TagService
	playlists *services.PlaylistService
	teams     *services.TeamService
	users     *services.UserService
}

func NewHandler(authMW *middleware.AuthMiddleware, auth *services.AuthService, songs *services.SongService, tags *services.TagService, playlists *services.PlaylistService, teams *services.TeamService, users *services.UserService) *Handler {
	return &Handler{authMW: authMW, auth: auth, songs: songs, tags: tags, playlists: playlists, teams: teams, users: users}
}

func (h *Handler) Register(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return utils.OK(c, 200, "Welcome to Songbanks API (Go)", fiber.Map{"version": "2.0.0-go"})
	})

	api := app.Group("/api")
	api.Post("/auth/login", h.Login)
	api.Get("/auth/google", h.GoogleLogin)
	api.Get("/auth/google/callback", h.GoogleCallback)
	api.Get("/artists", h.GetArtists)
	api.Get("/songs", h.GetSongs)
	api.Get("/songs/:id", h.GetSongByID)
	api.Get("/tags", h.GetTags)
	api.Post("/tags/get-or-create", h.GetOrCreateTag)

	auth := api.Group("", h.authMW.RequireAuth)
	auth.Post("/auth/logout", func(c *fiber.Ctx) error { return utils.OK(c, 200, "Logout successful", nil) })
	auth.Get("/auth/me", h.GetMe)
	auth.Get("/auth/check-permission", h.CheckPermission)

	auth.Post("/playlists", h.CreatePlaylist)
	auth.Get("/playlists", h.GetPlaylists)
	auth.Get("/playlists/:id", h.GetPlaylistByID)
	auth.Put("/playlists/:id", h.UpdatePlaylist)
	auth.Delete("/playlists/:id", h.DeletePlaylist)
	auth.Post("/playlists/:id/sharelink", h.GenerateSharelink)
	auth.Post("/playlists/join/:shareToken", h.JoinPlaylist)
	auth.Post("/playlists/:id/songs", h.AddSongsToPlaylist)
	auth.Post("/playlists/:id/songs/:songId", h.AddSongToPlaylistWithBaseChord)
	auth.Delete("/playlists/:id/song/:songId", h.RemoveSongFromPlaylist)

	auth.Get("/playlist-teams", h.GetMyTeams)
	auth.Get("/playlist-teams/:id", h.GetTeamByID)
	auth.Delete("/playlist-teams/:id/members/:user_id", h.RemoveMember)
	auth.Delete("/playlist-teams/:id", h.DeleteTeam)
	auth.Post("/playlist-teams/:id/leave", h.LeaveTeam)

	admin := auth.Group("/admin", h.authMW.RequirePengurus)
	admin.Post("/songs", h.CreateSong)
	admin.Put("/songs/:id", h.UpdateSong)
	admin.Delete("/songs/:id", h.DeleteSong)
	admin.Get("/user/", h.GetUsers)

	auth.Post("/notes/:user_id/:song_id", h.NotesUnavailable)
	auth.Get("/notes/:user_id", h.NotesUnavailable)
	auth.Get("/notes/:user_id/:id", h.NotesUnavailable)
	auth.Put("/notes/:user_id/:id", h.NotesUnavailable)
	auth.Delete("/notes/:user_id/:id", h.NotesUnavailable)

	app.Use(func(c *fiber.Ctx) error {
		return utils.Fail(c, 404, fmt.Sprintf("Route %s %s not found", c.Method(), c.OriginalURL()))
	})
}

func parseID(c *fiber.Ctx, key string) (int, error) {
	v, err := strconv.Atoi(c.Params(key))
	if err != nil {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return v, nil
}
