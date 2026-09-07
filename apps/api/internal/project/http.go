package project

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"taskline/apps/api/internal/auth"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authentication *auth.Service) http.Handler {
	handler := &Handler{service: service, auth: authentication}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{workspaceID}", handler.list)
	mux.HandleFunc("POST /{workspaceID}", handler.create)
	mux.HandleFunc("GET /{workspaceID}/{slug}", handler.get)
	mux.HandleFunc("PATCH /{workspaceID}/{slug}", handler.update)
	return authentication.Authenticate(mux)
}

type projectRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	projects, err := h.service.List(r.Context(), identity(r).UserID, workspaceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	var request projectRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	created, err := h.service.Create(r.Context(), identity(r).UserID, workspaceID, CreateInput{Name: request.Name, Slug: request.Slug, Description: request.Description})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"project": created})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	project, err := h.service.Get(r.Context(), identity(r).UserID, workspaceID, r.PathValue("slug"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	var request projectRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	updated, err := h.service.Update(r.Context(), identity(r).UserID, workspaceID, r.PathValue("slug"), CreateInput{Name: request.Name, Slug: request.Slug, Description: request.Description}, request.Archived)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": updated})
}

func identity(r *http.Request) auth.Identity {
	identity, _ := auth.IdentityFromContext(r.Context())
	return identity
}
func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return uuid.Nil, false
	}
	return id, true
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You do not have permission to do that")
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "project_not_found", "Project not found")
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
	case errors.Is(err, ErrSlugExists):
		writeError(w, http.StatusConflict, "project_slug_exists", "A project with this URL name already exists")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
