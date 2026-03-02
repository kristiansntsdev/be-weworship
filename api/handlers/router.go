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
audit     *services.AuditService
}

func NewHandler(authMW *middleware.AuthMiddleware, auth *services.AuthService, songs *services.SongService, tags *services.TagService, playlists *services.PlaylistService, teams *services.TeamService, users *services.UserService, analytics *services.AnalyticsService, audit *services.AuditService) *Handler {
return &Handler{authMW: authMW, auth: auth, songs: songs, tags: tags, playlists: playlists, teams: teams, users: users, analytics: analytics, audit: audit}
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
// NOTE: Do NOT use api.Group("", middleware) — in Fiber v2, an empty-prefix
// group registers the middleware as a global Use() on all /api/* routes,
// including public ones. Apply middleware inline per route instead.
ra := h.authMW.RequireAuth

api.Post("/auth/logout", ra, func(c *fiber.Ctx) error { return utils.OK(c, 200, "Logout successful", nil) })
api.Get("/auth/me", ra, h.GetMe)
api.Get("/auth/check-permission", ra, h.CheckPermission)

api.Post("/playlists", ra, h.CreatePlaylist)
api.Get("/playlists", ra, h.GetPlaylists)
api.Get("/playlists/:id", ra, h.GetPlaylistByID)
api.Put("/playlists/:id", ra, h.UpdatePlaylist)
api.Delete("/playlists/:id", ra, h.DeletePlaylist)
api.Post("/playlists/:id/sharelink", ra, h.GenerateSharelink)
api.Post("/playlists/join/:shareToken", ra, h.JoinPlaylist)
api.Post("/playlists/:id/songs", ra, h.AddSongsToPlaylist)
api.Post("/playlists/:id/songs/:songId", ra, h.AddSongToPlaylistWithBaseChord)
api.Delete("/playlists/:id/song/:songId", ra, h.RemoveSongFromPlaylist)

api.Get("/playlist-teams", ra, h.GetMyTeams)
api.Get("/playlist-teams/:id", ra, h.GetTeamByID)
api.Delete("/playlist-teams/:id/members/:user_id", ra, h.RemoveMember)
api.Delete("/playlist-teams/:id", ra, h.DeleteTeam)
api.Post("/playlist-teams/:id/leave", ra, h.LeaveTeam)

// ── Admin routes ───────────────────────────────────────────────────────
// Song CRUD: accessible by admin or maintainer
rm := h.authMW.RequireMaintainer
api.Post("/admin/songs", ra, rm, h.CreateSong)
api.Put("/admin/songs/:id", ra, rm, h.UpdateSong)
api.Delete("/admin/songs/:id", ra, rm, h.DeleteSong)
admin := api.Group("/admin", ra, h.authMW.RequireAdmin)
admin.Get("/users", h.GetUsers)

// Analytics (admin read)
admin.Get("/analytics/songs", h.GetAnalyticsSongs)
admin.Get("/analytics/users", h.GetAnalyticsUsers)
admin.Get("/analytics/searches", h.GetAnalyticsSearches)
admin.Get("/analytics/sessions", h.GetAnalyticsSessions)
admin.Get("/analytics/performance", h.GetAnalyticsPerformance)

// Audit log
admin.Get("/audit-logs", h.GetAuditLogs)

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
