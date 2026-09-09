package invite

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"taskline/apps/api/internal/auth"
	"taskline/apps/api/internal/workspace"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authentication *auth.Service) http.Handler {
	h := &Handler{service: service, auth: authentication}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /workspaces/{workspaceID}", h.create)
	mux.HandleFunc("GET /{token}", h.preview)
	mux.HandleFunc("POST /{token}/accept", h.accept)
	return authentication.Authenticate(mux)
}

type createRequest struct {
	Email string         `json:"email"`
	Role  workspace.Role `json:"role"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(r.PathValue("workspaceID"))
	if err != nil {
		writeError(w, ErrInvalidRequest)
		return
	}
	var request createRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	invite, err := h.service.Create(r.Context(), identity.UserID, workspaceID, request.Email, request.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invite": invite})
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	preview, err := h.service.Preview(r.Context(), r.PathValue("token"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite": preview})
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	membership, err := h.service.AcceptForUser(r.Context(), r.PathValue("token"), identity.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"membership": membership})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrEmailMismatch) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "forbidden", "message": "You do not have permission to do that"}})
		return
	}
	if errors.Is(err, ErrUnavailable) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "invite_not_found", "message": "Invitation is unavailable"}})
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "forbidden", "message": "You do not have permission to do that"}})
		return
	}
	if errors.Is(err, ErrInvalidRequest) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_request", "message": "Invalid request"}})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "Internal server error"}})
}

func pathUUID(value string) (uuid.UUID, error) { return uuid.Parse(value) }
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(&struct{}{}) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_request", "message": "Invalid request body"}})
		return false
	}
	return true
}

var _ workspace.Role
