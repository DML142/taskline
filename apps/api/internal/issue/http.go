package issue

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
}

func NewHandler(service *Service, authentication *auth.Service) http.Handler {
	handler := &Handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{workspaceID}/{projectSlug}", handler.list)
	mux.HandleFunc("POST /{workspaceID}/{projectSlug}", handler.create)
	mux.HandleFunc("GET /{workspaceID}/{projectSlug}/{issueID}", handler.get)
	mux.HandleFunc("PATCH /{workspaceID}/{projectSlug}/{issueID}", handler.update)
	return authentication.Authenticate(mux)
}

type issueRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assigneeId"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, ok := pathIDs(w, r, false)
	if !ok || issueID != uuid.Nil {
		return
	}
	filter, err := listFilter(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	issues, err := h.service.List(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	workspaceID, _, ok := pathIDs(w, r, false)
	if !ok {
		return
	}
	var request issueRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	created, err := h.service.Create(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), CreateInput(request))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"issue": created})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, ok := pathIDs(w, r, true)
	if !ok {
		return
	}
	issue, err := h.service.Get(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), issueID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issue": issue})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	workspaceID, issueID, ok := pathIDs(w, r, true)
	if !ok {
		return
	}
	var request issueRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	updated, err := h.service.Update(r.Context(), identity(r).UserID, workspaceID, r.PathValue("projectSlug"), issueID, UpdateInput(request))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issue": updated})
}

func listFilter(r *http.Request) (ListFilter, error) {
	filter := ListFilter{
		Status:   Status(r.URL.Query().Get("status")),
		Priority: Priority(r.URL.Query().Get("priority")),
	}
	if filter.Priority != "" && !validPriority(filter.Priority) {
		return ListFilter{}, ErrInvalidRequest
	}
	if assignee := r.URL.Query().Get("assigneeId"); assignee != "" {
		assigneeID, err := uuid.Parse(assignee)
		if err != nil {
			return ListFilter{}, ErrInvalidRequest
		}
		filter.AssigneeID = &assigneeID
	}
	return filter, nil
}

func pathIDs(w http.ResponseWriter, r *http.Request, withIssue bool) (uuid.UUID, uuid.UUID, bool) {
	workspaceID, err := uuid.Parse(r.PathValue("workspaceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return uuid.Nil, uuid.Nil, false
	}
	if !withIssue {
		return workspaceID, uuid.Nil, true
	}
	issueID, err := uuid.Parse(r.PathValue("issueID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return uuid.Nil, uuid.Nil, false
	}
	return workspaceID, issueID, true
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
		writeError(w, http.StatusNotFound, "issue_not_found", "Issue not found")
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidAssignee):
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
