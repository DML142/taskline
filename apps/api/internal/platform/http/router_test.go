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

func TestRouterMountsWorkspaceRoutesBelowAPIPrefix(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	workspaces := http.NewServeMux()
	workspaces.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	response := httptest.NewRecorder()

	NewRouter(logger, fakePinger{}, nil, workspaces).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil))

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestRouterAddsCORSHeadersToNonAuthRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	workspaces := http.NewServeMux()
	workspaces.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/workspaces", nil)
	request.Header.Set("Origin", "https://app.taskline.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "Authorization")
	response := httptest.NewRecorder()

	NewRouterWithCORS(logger, fakePinger{}, "https://app.taskline.example", nil, workspaces).ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "https://app.taskline.example", response.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", response.Header().Get("Access-Control-Allow-Credentials"))
	require.Equal(t, "GET, POST, OPTIONS", response.Header().Get("Access-Control-Allow-Methods"))
	require.Equal(t, "Authorization, Content-Type", response.Header().Get("Access-Control-Allow-Headers"))
}

func TestRouterRejectsUnexpectedOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	workspaces := http.NewServeMux()
	workspaces.HandleFunc("POST /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", nil)
	request.Header.Set("Origin", "https://malicious.example")
	response := httptest.NewRecorder()

	NewRouterWithCORS(logger, fakePinger{}, "https://app.taskline.example", nil, workspaces).ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.JSONEq(t, `{"error":{"code":"forbidden_origin","message":"Origin is not allowed"}}`, response.Body.String())
}

func TestRouterMountsIssueRoutesBelowAPIPrefix(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	issues := http.NewServeMux()
	issues.HandleFunc("GET /workspace/project", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	response := httptest.NewRecorder()

	NewRouter(logger, fakePinger{}, nil, nil, nil, issues).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/issues/workspace/project", nil))

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestRouterCombinesIssueAndCommentRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	issues := http.NewServeMux()
	issues.HandleFunc("GET /workspace/project/issue", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	comments := http.NewServeMux()
	comments.HandleFunc("GET /workspace/project/issue/comments", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })

	router := NewRouter(logger, fakePinger{}, nil, nil, nil, issues, nil, comments)

	issueResponse := httptest.NewRecorder()
	router.ServeHTTP(issueResponse, httptest.NewRequest(http.MethodGet, "/api/v1/issues/workspace/project/issue", nil))
	commentResponse := httptest.NewRecorder()
	router.ServeHTTP(commentResponse, httptest.NewRequest(http.MethodGet, "/api/v1/issues/workspace/project/issue/comments", nil))

	require.Equal(t, http.StatusNoContent, issueResponse.Code)
	require.Equal(t, http.StatusAccepted, commentResponse.Code)
}
