package platform

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Context struct {
	DB        *sqlx.DB
	JWTSecret []byte
	ClientURL string
}

func NewContext() (*Context, error) {
	dsn := buildDSN()

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Context{
		DB:        db,
		JWTSecret: []byte(env("SESSION_SECRET", "dev-secret")),
		ClientURL: env("CLIENT_URL", "http://localhost:3000"),
	}, nil
}

func buildDSN() string {
	if rawURL := strings.TrimSpace(os.Getenv("PROD_DB_URL")); rawURL != "" {
		if dsn, err := dsnFromURL(rawURL); err == nil {
			return dsn
		}
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		env("PROD_DB_USERNAME", "songbank"),
		env("PROD_DB_PASSWORD", "songbank"),
		env("PROD_DB_HOST", "127.0.0.1"),
		normalizePort(env("PROD_DB_PORT", "3306")),
		env("PROD_DB_DATABASE", "songbanksdb"),
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
		port = "3306"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", user, pass, host, port, dbName), nil
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
