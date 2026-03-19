package handler

import (
	"net/http"
	"os"
	"sync"

	"be-songbanks-v1/api/handlers"
	"be-songbanks-v1/api/middleware"
	"be-songbanks-v1/api/platform"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services"
	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	once    sync.Once
	fapp    *fiber.App
	initErr error
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		// Try common .env locations (local dev: api/.env, project root: .env)
		for _, p := range []string{"api/.env", ".env", "../api/.env"} {
			if err := godotenv.Load(p); err == nil {
				break
			}
		}
		fapp, initErr = buildApp()
	})
	if initErr != nil {
		http.Error(w, initErr.Error(), http.StatusInternalServerError)
		return
	}
	// Vercel serverless rewrites r.RequestURI to the function file path (e.g. /api/index.go),
	// which strips the original path and query string. r.URL retains the original values.
	if r.URL != nil {
		r.RequestURI = r.URL.RequestURI()
	}
	adaptor.FiberApp(fapp)(w, r)
}

func buildApp() (*fiber.App, error) {
	ctx, err := platform.NewContext()
	if err != nil {
		return nil, err
	}

	authRepo := repositories.NewAuthRepository(ctx.DB)
	tagRepo := repositories.NewTagRepository(ctx.DB)
	songRepo := repositories.NewSongRepository(ctx.DB)
	playlistRepo := repositories.NewPlaylistRepository(ctx.DB)
	teamRepo := repositories.NewTeamRepository(ctx.DB)
	userRepo := repositories.NewUserRepository(ctx.DB)
	analyticsRepo := repositories.NewAnalyticsRepository(ctx.DB)

	songCache := platform.NewSongCache()
	liveCache := platform.NewLiveCache()

	authSvc := services.NewAuthService(authRepo, ctx.JWTSecret, services.GoogleConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("GOOGLE_REDIRECT_URI"),
		ClientURL:    os.Getenv("CLIENT_URL"),
		MobileScheme: os.Getenv("MOBILE_SCHEME"),
	})
	tagSvc := services.NewTagService(tagRepo)
	songSvc := services.NewSongService(songRepo, tagRepo, playlistRepo, songCache)
	playlistSvc := services.NewPlaylistService(playlistRepo, teamRepo, songRepo, ctx.ClientURL, liveCache)
	teamSvc := services.NewTeamService(teamRepo, authRepo, playlistRepo)
	userSvc := services.NewUserService(userRepo)
	analyticsSvc := services.NewAnalyticsService(analyticsRepo)
	auditRepo := repositories.NewAuditRepository(ctx.DB)
	auditSvc := services.NewAuditService(auditRepo)

	notifRepo := repositories.NewNotificationRepository(ctx.DB)
	notifSvc := services.NewNotificationService(ctx.FCM, notifRepo)

	authMW := middleware.NewAuthMiddleware(authSvc)
	h := handlers.NewHandler(authMW, authSvc, songSvc, tagSvc, playlistSvc, teamSvc, userSvc, analyticsSvc, auditSvc, notifSvc)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
	app.Options("/*", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path} ip=${ip} ua=${ua}\n",
	}))
	h.Register(app)
	return app, nil
}
