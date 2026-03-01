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
analytics *services.AnalyticsService
}

func NewHandler(authMW *middleware.AuthMiddleware, auth *services.AuthService, songs *services.SongService, tags *services.TagService, playlists *services.PlaylistService, teams *services.TeamService, users *services.UserService, analytics *services.AnalyticsService) *Handler {
return &Handler{authMW: authMW, auth: auth, songs: songs, tags: tags, playlists: playlists, teams: teams, users: users, analytics: analytics}
}

func (h *Handler) Register(app *fiber.App) {
app.Get("/", func(c *fiber.Ctx) error {
return utils.OK(c, 200, "Welcome to WeWorship API", fiber.Map{"version": "3.0.0"})
})

api := app.Group("/api")

// ── Public routes ──────────────────────────────────────────────────────
api.Post("/auth/register", h.RegisterUser)
api.Post("/auth/login", h.Login)
api.Get("/auth/google", h.GoogleLogin)
api.Get("/auth/google/callback", h.GoogleCallback)
api.Get("/home", h.GetHome)
api.Get("/artists", h.GetArtists)
api.Get("/songs", h.GetSongs)
api.Get("/songs/:id", h.GetSongByID)
api.Get("/tags", h.GetTags)
api.Post("/tags/get-or-create", h.GetOrCreateTag)

// ── Analytics write (optionally authenticated) ─────────────────────────
api.Post("/analytics/performance", h.RecordPerformance)
api.Post("/analytics/session", h.RecordSession)

// ── Auth-required routes ───────────────────────────────────────────────
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

// ── Admin routes ───────────────────────────────────────────────────────
admin := auth.Group("/admin", h.authMW.RequireAdmin)
admin.Post("/songs", h.CreateSong)
admin.Put("/songs/:id", h.UpdateSong)
admin.Delete("/songs/:id", h.DeleteSong)
admin.Get("/users", h.GetUsers)

// Analytics (admin read)
admin.Get("/analytics/songs", h.GetAnalyticsSongs)
admin.Get("/analytics/users", h.GetAnalyticsUsers)
admin.Get("/analytics/searches", h.GetAnalyticsSearches)
admin.Get("/analytics/sessions", h.GetAnalyticsSessions)
admin.Get("/analytics/performance", h.GetAnalyticsPerformance)

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

func parseInt(s string) (int, error) {
return strconv.Atoi(s)
}
