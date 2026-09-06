package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestRouter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(logger, fakePinger{}, nil)
	for _, tt := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"health", http.MethodGet, "/api/v1/health", `{"status":"ok"}`, http.StatusOK},
		{"missing route", http.MethodGet, "/api/v1/missing", `{"error":{"code":"not_found","message":"Resource not found"}}`, http.StatusNotFound},
		{"unsupported method", http.MethodPost, "/api/v1/health", `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`, http.StatusMethodNotAllowed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, "application/json", response.Header().Get("Content-Type"))
			require.NotEmpty(t, response.Header().Get("X-Request-ID"))
			require.JSONEq(t, tt.body, response.Body.String())
		})
	}
}

func TestRouterReady(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("returns ok when database is reachable", func(t *testing.T) {
		response := httptest.NewRecorder()
		NewRouter(logger, fakePinger{}, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil))

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "application/json", response.Header().Get("Content-Type"))
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		require.NotEmpty(t, response.Header().Get("X-Request-ID"))
		require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	})

	t.Run("reports database unavailable when ping fails", func(t *testing.T) {
		response := httptest.NewRecorder()
		NewRouter(logger, fakePinger{err: errors.New("connection refused")}, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil))

		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		require.Equal(t, "application/json", response.Header().Get("Content-Type"))
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		require.NotEmpty(t, response.Header().Get("X-Request-ID"))
		require.JSONEq(t, `{"error":{"code":"database_unavailable","message":"Database is unavailable"}}`, response.Body.String())
	})

	t.Run("health remains live when database ping fails", func(t *testing.T) {
		response := httptest.NewRecorder()
		NewRouter(logger, fakePinger{err: errors.New("connection refused")}, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

		require.Equal(t, http.StatusOK, response.Code)
		require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	})
}

func TestRouterMountsAuthenticationRoutesBelowAPIPrefix(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authentication := http.NewServeMux()
	authentication.HandleFunc("POST /register", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	response := httptest.NewRecorder()

	NewRouter(logger, fakePinger{}, authentication).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil))

	require.Equal(t, http.StatusCreated, response.Code)
}
