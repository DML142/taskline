package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
	config  Config
	limiter *authRateLimiter
}

func NewHandler(logger *slog.Logger, service *Service, config Config) *Handler {
	return &Handler{logger: logger, service: service, config: config, limiter: newAuthRateLimiter(time.Minute)}
}
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /verify-email", h.verifyEmail)
	mux.HandleFunc("POST /resend-verification", h.resendVerification)
	mux.HandleFunc("POST /refresh", h.refresh)
	mux.HandleFunc("POST /logout", h.logout)
	mux.HandleFunc("GET /me", h.me)
	return h.cors(mux)
}

type credentialsRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in credentialsRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if !h.limiter.allow(r, "register", in.Email) {
		writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "Too many authentication attempts")
		return
	}
	result, err := h.service.Register(r.Context(), RegisterInput(in))
	if err != nil {
		h.limiter.recordFailure(r, "register", in.Email)
		writeAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid registration details")
		return
	}
	_ = result
	writeJSON(w, http.StatusAccepted, map[string]bool{"verificationRequired": true})
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in credentialsRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if !h.limiter.allow(r, "login", in.Email) {
		writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "Too many authentication attempts")
		return
	}
	result, err := h.service.Login(r.Context(), LoginInput{Email: in.Email, Password: in.Password})
	if err != nil {
		if errors.Is(err, ErrEmailVerificationRequired) {
			writeAuthError(w, http.StatusForbidden, "email_verification_required", "Verify your email before signing in")
			return
		}
		h.limiter.recordFailure(r, "login", in.Email)
		writeAuthError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}
	h.writeAuthentication(w, result, http.StatusOK)
}

type verificationRequest struct {
	Token string `json:"token"`
}
type resendVerificationRequest struct {
	Email string `json:"email"`
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var in verificationRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.service.VerifyEmail(r.Context(), in.Token); err != nil {
		writeAuthError(w, http.StatusBadRequest, "verification_unavailable", "This verification link is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"verified": true})
}

func (h *Handler) resendVerification(w http.ResponseWriter, r *http.Request) {
	var in resendVerificationRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if !h.limiter.take(r, "resend-verification", in.Email) {
		writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "Too many authentication attempts")
		return
	}
	if err := h.service.ResendVerification(r.Context(), in.Email); err != nil {
		h.limiter.recordFailure(r, "resend-verification", in.Email)
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"verificationRequired": true})
}

type authRateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	window  time.Duration
}

const maxRateLimitKeys = 1024

func newAuthRateLimiter(window time.Duration) *authRateLimiter {
	return &authRateLimiter{entries: map[string][]time.Time{}, window: window}
}
func (l *authRateLimiter) allow(r *http.Request, endpoint, email string) bool {
	key := rateLimitKey(r, endpoint, email)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	return len(l.entries[key]) < 5
}

func (l *authRateLimiter) take(r *http.Request, endpoint, email string) bool {
	key := rateLimitKey(r, endpoint, email)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	if len(l.entries[key]) >= 5 {
		return false
	}
	if _, exists := l.entries[key]; !exists && len(l.entries) >= maxRateLimitKeys {
		return false
	}
	l.entries[key] = append(l.entries[key], now)
	return true
}

func (l *authRateLimiter) recordFailure(r *http.Request, endpoint, email string) {
	key := rateLimitKey(r, endpoint, email)
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.prune(now)
	if _, exists := l.entries[key]; !exists && len(l.entries) >= maxRateLimitKeys {
		return
	}
	l.entries[key] = append(l.entries[key], now)
}

func (l *authRateLimiter) prune(now time.Time) {
	for key, attempts := range l.entries {
		kept := attempts[:0]
		for _, attempt := range attempts {
			if now.Sub(attempt) < l.window {
				kept = append(kept, attempt)
			}
		}
		if len(kept) == 0 {
			delete(l.entries, key)
		} else {
			l.entries[key] = kept
		}
	}
}

func rateLimitKey(r *http.Request, endpoint, email string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.Join([]string{host, endpoint}, "\x00")
}
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("taskline_refresh")
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	result, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	h.writeAuthentication(w, result, http.StatusOK)
}
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("taskline_refresh"); err == nil {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	h.setRefreshCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == h.config.WebOrigin {
			w.Header().Set("Access-Control-Allow-Origin", h.config.WebOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user)})
}
func (h *Handler) writeAuthentication(w http.ResponseWriter, a Authentication, status int) {
	h.setRefreshCookie(w, a.RefreshToken, int(refreshSessionLifetime.Seconds()))
	writeJSON(w, status, map[string]any{"user": publicUser(a.User), "accessToken": a.AccessToken})
}
func (h *Handler) setRefreshCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: "taskline_refresh", Value: value, Path: "/api/v1/auth", HttpOnly: true, Secure: h.config.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
func publicUser(u StoredUser) map[string]string {
	return map[string]string{"id": u.ID.String(), "email": u.Email, "name": u.Name}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
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
func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
