package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHandlerRegisterRequiresEmailVerification(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	email := fmt.Sprintf("http-%d@example.com", time.Now().UnixNano())
	body, err := json.Marshal(credentialsRequest{Name: "Ada", Email: email, Password: validTestPassword})
	require.NoError(t, err)
	register := httptest.NewRecorder()
	handler.Routes().ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	require.Equal(t, http.StatusAccepted, register.Code)
	require.Empty(t, register.Header().Get("Set-Cookie"))
	require.JSONEq(t, `{"verificationRequired":true}`, register.Body.String())
}

func TestHandlerRejectsInvalidLogin(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"correct horse battery staple1"}`))
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, request)
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.JSONEq(t, `{"error":{"code":"invalid_credentials","message":"Invalid email or password"}}`, response.Body.String())
}

func TestHandlerAllowsConfiguredOriginOnly(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{WebOrigin: "http://localhost:3000"})
	allowed := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/login", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	handler.Routes().ServeHTTP(allowed, request)
	require.Equal(t, http.StatusNoContent, allowed.Code)
	require.Equal(t, "http://localhost:3000", allowed.Header().Get("Access-Control-Allow-Origin"))

	denied := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodOptions, "/login", nil)
	request.Header.Set("Origin", "https://attacker.example")
	handler.Routes().ServeHTTP(denied, request)
	require.Empty(t, denied.Header().Get("Access-Control-Allow-Origin"))
}

func TestHandlerRateLimitsAuthenticationAttempts(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	for range 5 {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"correct horse battery staple1"}`))
		request.RemoteAddr = "198.51.100.1:1234"
		handler.Routes().ServeHTTP(response, request)
		require.Equal(t, http.StatusUnauthorized, response.Code)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"correct horse battery staple1"}`))
	request.RemoteAddr = "198.51.100.1:5678"
	handler.Routes().ServeHTTP(response, request)
	require.Equal(t, http.StatusTooManyRequests, response.Code)
}

func TestHandlerRateLimitsSuccessfulVerificationResends(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	email := fmt.Sprintf("resend-%d@example.com", time.Now().UnixNano())
	registration := httptest.NewRecorder()
	registrationRequest := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(fmt.Sprintf(`{"name":"Resend user","email":%q,"password":"correct horse battery staple1"}`, email)))
	registrationRequest.RemoteAddr = "198.51.100.4:1234"
	handler.Routes().ServeHTTP(registration, registrationRequest)
	require.Equal(t, http.StatusAccepted, registration.Code)

	for range 5 {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/resend-verification", bytes.NewBufferString(fmt.Sprintf(`{"email":%q}`, email)))
		request.RemoteAddr = "198.51.100.4:5678"
		handler.Routes().ServeHTTP(response, request)
		require.Equal(t, http.StatusAccepted, response.Code)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/resend-verification", bytes.NewBufferString(fmt.Sprintf(`{"email":%q}`, email)))
	request.RemoteAddr = "198.51.100.4:4321"
	handler.Routes().ServeHTTP(response, request)
	require.Equal(t, http.StatusTooManyRequests, response.Code)
}

func TestAuthRateLimiterTakeCountsEveryAllowedAttempt(t *testing.T) {
	limiter := newAuthRateLimiter(time.Minute)
	request := httptest.NewRequest(http.MethodPost, "/resend-verification", nil)
	request.RemoteAddr = "198.51.100.5:1234"
	for range 5 {
		require.True(t, limiter.take(request, "resend-verification", "ada@example.com"))
	}
	require.False(t, limiter.take(request, "resend-verification", "ada@example.com"))
}

func TestHandlerDoesNotRateLimitSuccessfulRegistrationThenLogin(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	email := fmt.Sprintf("fresh-%d@example.com", time.Now().UnixNano())
	register := httptest.NewRecorder()
	registerRequest := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(fmt.Sprintf(`{"name":"Fresh user","email":%q,"password":"correct horse battery staple1"}`, email)))
	registerRequest.RemoteAddr = "198.51.100.2:1234"
	handler.Routes().ServeHTTP(register, registerRequest)
	require.Equal(t, http.StatusAccepted, register.Code)
	user, err := handler.service.repository.FindUserByEmail(context.Background(), email)
	require.NoError(t, err)
	verifyTestUser(t, handler.service, user.ID)

	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple1"}`, email)))
	loginRequest.RemoteAddr = "198.51.100.2:5678"
	handler.Routes().ServeHTTP(login, loginRequest)
	require.Equal(t, http.StatusOK, login.Code)
}

func TestRateLimitKeyDoesNotVaryByEmail(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.RemoteAddr = "198.51.100.3:1234"
	require.Equal(t, rateLimitKey(request, "login", "ada@example.com"), rateLimitKey(request, "login", "grace@example.com"))
}
