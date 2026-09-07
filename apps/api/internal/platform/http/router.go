package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Pinger interface {
	Ping(context.Context) error
}

func NewRouter(logger *slog.Logger, readiness Pinger, authentication http.Handler, workspaces ...http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			ww.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
			defer func() {
				logger.InfoContext(r.Context(), "http request",
					"request_id", middleware.GetReqID(r.Context()),
					"method", r.Method, "path", r.URL.Path,
					"status", ww.Status(), "duration", time.Since(start))
			}()
			next.ServeHTTP(ww, r)
		})
	})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == http.ErrAbortHandler {
						panic(recovered)
					}
					logger.ErrorContext(r.Context(), "request panic", "request_id", middleware.GetReqID(r.Context()))
					writeError(logger, w, http.StatusInternalServerError, "internal_error", "Internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, err := w.Write([]byte("{\"status\":\"ok\"}\n"))
		if err != nil {
			logger.Debug("write health response", "error", err)
		}
	})
	r.Get("/api/v1/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if err := readiness.Ping(r.Context()); err != nil {
			logger.WarnContext(r.Context(), "database readiness check failed", "request_id", middleware.GetReqID(r.Context()))
			writeError(logger, w, http.StatusServiceUnavailable, "database_unavailable", "Database is unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\":\"ok\"}\n")); err != nil {
			logger.Debug("write readiness response", "error", err)
		}
	})
	if authentication != nil {
		r.Mount("/api/v1/auth", http.StripPrefix("/api/v1/auth", authentication))
	}
	if len(workspaces) > 0 && workspaces[0] != nil {
		workspaceRoutes := http.StripPrefix("/api/v1/workspaces", workspaces[0])
		workspaceRoot := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			request.URL.Path = "/"
			workspaces[0].ServeHTTP(w, request)
		})
		r.Method(http.MethodGet, "/api/v1/workspaces", workspaceRoot)
		r.Method(http.MethodPost, "/api/v1/workspaces", workspaceRoot)
		r.Mount("/api/v1/workspaces/", workspaceRoutes)
	}
	if len(workspaces) > 1 && workspaces[1] != nil {
		projectRoutes := http.StripPrefix("/api/v1/projects", workspaces[1])
		r.Mount("/api/v1/projects/", projectRoutes)
	}
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(logger, w, http.StatusNotFound, "not_found", "Resource not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(logger, w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	})
	return r
}
