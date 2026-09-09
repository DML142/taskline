package config

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskline/apps/api/internal/auth"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	Auth        auth.Config
	SMTP        SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func Load() (Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_ADDR must be a host:port address: %w", err)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("HTTP_ADDR port must be between 1 and 65535")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	parsedDatabaseURL, err := url.Parse(databaseURL)
	if err != nil || (parsedDatabaseURL.Scheme != "postgres" && parsedDatabaseURL.Scheme != "postgresql") {
		return Config{}, fmt.Errorf("DATABASE_URL must be a valid PostgreSQL connection string")
	}
	if _, err := pgxpool.ParseConfig(databaseURL); err != nil {
		return Config{}, fmt.Errorf("DATABASE_URL must be a valid PostgreSQL connection string")
	}
	authConfig, err := loadAuthConfig()
	if err != nil {
		return Config{}, err
	}
	smtpConfig, err := loadSMTPConfig()
	if err != nil {
		return Config{}, err
	}
	return Config{HTTPAddr: addr, DatabaseURL: databaseURL, Auth: authConfig, SMTP: smtpConfig}, nil
}

func loadSMTPConfig() (SMTPConfig, error) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP_HOST is required")
	}
	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil || port < 1 || port > 65535 {
		return SMTPConfig{}, fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	username := os.Getenv("SMTP_USERNAME")
	if username == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP_USERNAME is required")
	}
	password := os.Getenv("SMTP_PASSWORD")
	if password == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP_PASSWORD is required")
	}
	from := os.Getenv("SMTP_FROM")
	parsedFrom, err := mail.ParseAddress(from)
	if err != nil || parsedFrom.Address == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	return SMTPConfig{Host: host, Port: port, Username: username, Password: password, From: from}, nil
}

func loadAuthConfig() (auth.Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return auth.Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "taskline-api"
	}
	audience := os.Getenv("JWT_AUDIENCE")
	if audience == "" {
		audience = "taskline-web"
	}
	webOrigin := os.Getenv("WEB_ORIGIN")
	if webOrigin == "" {
		webOrigin = "http://localhost:3000"
	}
	parsedOrigin, err := url.Parse(webOrigin)
	if err != nil || !parsedOrigin.IsAbs() || parsedOrigin.Host == "" || parsedOrigin.User != nil || (parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https") || parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" {
		return auth.Config{}, fmt.Errorf("WEB_ORIGIN must be an absolute HTTP(S) origin")
	}
	cookieSecure := false
	if rawCookieSecure := os.Getenv("COOKIE_SECURE"); rawCookieSecure != "" {
		cookieSecure, err = strconv.ParseBool(rawCookieSecure)
		if err != nil {
			return auth.Config{}, fmt.Errorf("COOKIE_SECURE must be a boolean")
		}
	}
	return auth.Config{
		JWTSecret:    []byte(secret),
		JWTIssuer:    issuer,
		JWTAudience:  audience,
		WebOrigin:    webOrigin,
		CookieSecure: cookieSecure,
	}, nil
}
