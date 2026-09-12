package comment

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"taskline/apps/api/internal/auth"
)

type Handler struct{ service *Service }

func NewHandler(service *Service, authentication *auth.Service) http.Handler {
	handler := &Handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{workspaceID}/{projectSlug}/{issueID}/comments", handler.list)
	mux.HandleFunc("POST /{workspaceID}/{projectSlug}/{issueID}/comments", handler.create)
	mux.HandleFunc("PATCH /{workspaceID}/{projectSlug}/{issueID}/comments/{commentID}", handler.update)
	mux.HandleFunc("DELETE /{workspaceID}/{projectSlug}/{issueID}/comments/{commentID}", handler.delete)
	return authentication.Authenticate(mux)
}

type commentRequest struct {
	Body string `json:"body"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, _, ok := pathIDs(w, r, false)
	if !ok {
		return
	}
	comments, err := h.service.List(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), issueID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, _, ok := pathIDs(w, r, false)
	if !ok {
		return
	}
	var request commentRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	comment, err := h.service.Create(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), issueID, CreateInput(request))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"comment": comment})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, commentID, ok := pathIDs(w, r, true)
	if !ok {
		return
	}
	var request commentRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	comment, err := h.service.Update(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), issueID, commentID, UpdateInput(request))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comment": comment})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, commentID, ok := pathIDs(w, r, true)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), issueID, commentID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathIDs(w http.ResponseWriter, r *http.Request, withComment bool) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	workspaceID, err := uuid.Parse(r.PathValue("workspaceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	issueID, err := uuid.Parse(r.PathValue("issueID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if !withComment {
		return workspaceID, issueID, uuid.Nil, true
	}
	commentID, err := uuid.Parse(r.PathValue("commentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return workspaceID, issueID, commentID, true
}

func identity(r *http.Request) auth.Identity {
	identity, _ := auth.IdentityFromContext(r.Context())
	return identity
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
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "comment_not_found", "Comment not found")
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
