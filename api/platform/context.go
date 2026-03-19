package platform

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"be-songbanks-v1/api/providers"
	"github.com/jmoiron/sqlx"
)

type Context struct {
	DB        *sqlx.DB
	JWTSecret []byte
	ClientURL string
	FCM       *providers.FCMProvider
}

func NewContext() (*Context, error) {
	dsn := buildDSN()

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	ctx := &Context{
		DB:        db,
		JWTSecret: []byte(env("SESSION_SECRET", "dev-secret")),
		ClientURL: env("CLIENT_URL", "http://localhost:3000"),
	}

	// Initialise FCM provider (optional – skipped if env vars are missing)
	fcmProjectID := env("FCM_PROJECT_ID", "")
	fcmCredPath := env("FCM_CREDENTIALS_PATH", "")
	if fcmProjectID != "" && fcmCredPath != "" {
		fcmProvider, fcmErr := providers.NewFCMProvider(fcmProjectID, fcmCredPath)
		if fcmErr != nil {
			log.Printf("[fcm] provider init skipped: %v", fcmErr)
		} else {
			ctx.FCM = fcmProvider
		}
	} else {
		log.Println("[fcm] FCM_PROJECT_ID or FCM_CREDENTIALS_PATH not set – FCM disabled")
	}

	return ctx, nil
}

func buildDSN() string {
	if rawURL := strings.TrimSpace(os.Getenv("DB_URL")); rawURL != "" {
		if dsn, err := dsnFromURL(rawURL); err == nil {
			return dsn
		}
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env("DB_USERNAME", "songbank"),
		env("DB_PASSWORD", "songbank"),
		env("DB_HOST", "127.0.0.1"),
		normalizePort(env("DB_PORT", "5432")),
		env("DB_DATABASE", "songbanksdb"),
	)
}

func dsnFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	user := ""
	pass := ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	host := u.Hostname()
	port := normalizePort(u.Port())
	if port == "" {
		port = "5432"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbName, sslmode), nil
}

func normalizePort(port string) string {
	port = strings.TrimSpace(port)
	port = strings.TrimPrefix(port, "tcp/")
	return port
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
