package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	for _, tt := range []struct {
		value, want     string
		databaseURL     string
		wantDatabaseURL string
		invalid         bool
	}{
		{"", "127.0.0.1:8080", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", false},
		{":9090", ":9090", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", false},
		{"[::1]:8080", "[::1]:8080", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", false},
		{"localhost", "", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "", true},
		{":0", "", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "", true},
		{":70000", "", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "", true},
		{":http", "", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable", "", true},
		{"", "", "", "", true},
		{"", "", "http://localhost:5432/taskline", "", true},
	} {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", tt.value)
			t.Setenv("DATABASE_URL", tt.databaseURL)
			setValidAuthEnv(t)
			cfg, err := Load()
			if tt.invalid {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "pass")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, cfg.HTTPAddr)
			require.Equal(t, tt.wantDatabaseURL, cfg.DatabaseURL)
		})
	}
}

func TestLoadRejectsNonURLDatabaseConfig(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "host=127.0.0.1 port=5432 user=taskline dbname=taskline")
	setValidAuthEnv(t)

	_, err := Load()

	require.EqualError(t, err, "DATABASE_URL must be a valid PostgreSQL connection string")
}

func TestLoadDoesNotExposeMalformedDatabaseURL(t *testing.T) {
	const databaseURL = "postgres://user:unique-db-test-secret@127.0.0.1:5432/taskline?sslmode=not-a-valid-mode"
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", databaseURL)
	setValidAuthEnv(t)

	_, err := Load()

	require.EqualError(t, err, "DATABASE_URL must be a valid PostgreSQL connection string")
	require.NotContains(t, err.Error(), "unique-db-test-secret")
	require.NotContains(t, err.Error(), databaseURL)
}

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable")
	t.Setenv("JWT_SECRET", "too-short")

	_, err := Load()

	require.EqualError(t, err, "JWT_SECRET must be at least 32 bytes")
	require.NotContains(t, err.Error(), "too-short")
}

func TestLoadAcceptsAuthenticationConfiguration(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable")
	setValidAuthEnv(t)
	setValidSMTPEnv(t)
	t.Setenv("WEB_ORIGIN", "https://app.taskline.example")
	t.Setenv("COOKIE_SECURE", "true")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, "https://app.taskline.example", cfg.Auth.WebOrigin)
	require.True(t, cfg.Auth.CookieSecure)
	require.Equal(t, "taskline-api", cfg.Auth.JWTIssuer)
	require.Equal(t, "taskline-web", cfg.Auth.JWTAudience)
	require.Equal(t, "smtp.taskline.example", cfg.SMTP.Host)
	require.Equal(t, "starttls", cfg.SMTP.TLSMode)
}

func TestLoadAcceptsLocalSMTPWithoutTLS(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable")
	setValidAuthEnv(t)
	t.Setenv("SMTP_TLS_MODE", "none")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, "none", cfg.SMTP.TLSMode)
}

func TestLoadRequiresSMTPConfiguration(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@127.0.0.1:5432/taskline?sslmode=disable")
	setValidAuthEnv(t)
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")

	_, err := Load()

	require.EqualError(t, err, "SMTP_HOST is required")
}

func setValidAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))
	t.Setenv("JWT_ISSUER", "taskline-api")
	t.Setenv("JWT_AUDIENCE", "taskline-web")
	t.Setenv("WEB_ORIGIN", "http://localhost:3000")
	t.Setenv("COOKIE_SECURE", "false")
	setValidSMTPEnv(t)
}

func setValidSMTPEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SMTP_HOST", "smtp.taskline.example")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "taskline")
	t.Setenv("SMTP_PASSWORD", "smtp-test-password")
	t.Setenv("SMTP_FROM", "Taskline <no-reply@taskline.example>")
}
