package config

import (
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

	_, err := Load()

	require.EqualError(t, err, "DATABASE_URL must be a valid PostgreSQL connection string")
}

func TestLoadDoesNotExposeMalformedDatabaseURL(t *testing.T) {
	const databaseURL = "postgres://user:unique-db-test-secret@127.0.0.1:5432/taskline?sslmode=not-a-valid-mode"
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", databaseURL)

	_, err := Load()

	require.EqualError(t, err, "DATABASE_URL must be a valid PostgreSQL connection string")
	require.NotContains(t, err.Error(), "unique-db-test-secret")
	require.NotContains(t, err.Error(), databaseURL)
}
