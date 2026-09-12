package workspace

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"taskline/apps/api/internal/auth"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authentication *auth.Service) http.Handler {
	h := &Handler{service: service, auth: authentication}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.list)
	mux.HandleFunc("POST /", h.create)
	mux.HandleFunc("GET /{workspaceID}", h.get)
	mux.HandleFunc("PATCH /{workspaceID}", h.rename)
	mux.HandleFunc("DELETE /{workspaceID}", h.delete)
	mux.HandleFunc("GET /{workspaceID}/members", h.listMembers)
	mux.HandleFunc("PATCH /{workspaceID}/members/{userID}", h.changeMemberRole)
	mux.HandleFunc("DELETE /{workspaceID}/members/{userID}", h.removeMember)
	return authentication.Authenticate(mux)
}

type workspaceRequest struct {
	Name string `json:"name"`
}
type roleRequest struct {
	Role Role `json:"role"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), identity(r).UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request workspaceRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	workspace, err := h.service.Create(r.Context(), identity(r).UserID, request.Name)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"workspace": workspace.Workspace, "role": workspace.Role})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	workspace, err := h.service.Get(r.Context(), identity(r).UserID, workspaceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace.Workspace, "role": workspace.Role})
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	var request workspaceRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	workspace, err := h.service.Rename(r.Context(), identity(r).UserID, workspaceID, request.Name)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), identity(r).UserID, workspaceID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	members, err := h.service.ListMembers(r.Context(), identity(r).UserID, workspaceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h *Handler) changeMemberRole(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	userID, ok := pathUUID(w, r, "userID")
	if !ok {
		return
	}
	var request roleRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := h.service.ChangeMemberRole(r.Context(), identity(r).UserID, workspaceID, userID, request.Role); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := pathUUID(w, r, "workspaceID")
	if !ok {
		return
	}
	userID, ok := pathUUID(w, r, "userID")
	if !ok {
		return
	}
	if err := h.service.RemoveMember(r.Context(), identity(r).UserID, workspaceID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrWorkspaceNotFound):
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found")
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You do not have permission to do that")
	case errors.Is(err, ErrMemberExists):
		writeError(w, http.StatusConflict, "member_exists", "User is already a member")
	case errors.Is(err, ErrSlugExists):
		writeError(w, http.StatusConflict, "workspace_slug_exists", "A workspace with this URL name already exists")
	case errors.Is(err, ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user_not_found", "User not found")
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
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
