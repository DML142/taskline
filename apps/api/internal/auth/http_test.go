package auth

import (
	"bytes"
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

func TestHandlerRegisterAndMe(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	email := fmt.Sprintf("http-%d@example.com", time.Now().UnixNano())
	body, err := json.Marshal(credentialsRequest{Name: "Ada", Email: email, Password: validTestPassword})
	require.NoError(t, err)
	register := httptest.NewRecorder()
	handler.Routes().ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	require.Equal(t, http.StatusCreated, register.Code)
	require.Contains(t, register.Header().Get("Set-Cookie"), "HttpOnly")

	var response struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.Unmarshal(register.Body.Bytes(), &response))
	me := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer "+response.AccessToken)
	handler.Routes().ServeHTTP(me, request)
	require.Equal(t, http.StatusOK, me.Code)
}

func TestHandlerRejectsInvalidLogin(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), newTestService(t), Config{})
	request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"correct horse battery staple"}`))
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
	first := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"correct horse battery staple"}`))
	request.RemoteAddr = "198.51.100.1:1234"
	handler.Routes().ServeHTTP(first, request)
	require.Equal(t, http.StatusUnauthorized, first.Code)
	second := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"correct horse battery staple"}`))
	request.RemoteAddr = "198.51.100.1:5678"
	handler.Routes().ServeHTTP(second, request)
	require.Equal(t, http.StatusTooManyRequests, second.Code)
}
