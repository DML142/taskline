package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuthenticateAddsIdentityForAnActiveSession(t *testing.T) {
	service := newTestService(t)
	email := fmt.Sprintf("middleware-%d@example.com", time.Now().UnixNano())
	registration, err := service.Register(context.Background(), RegisterInput{
		Name:     "Middleware user",
		Email:    email,
		Password: validTestPassword,
	})
	require.NoError(t, err)
	verifyTestUser(t, service, registration.User.ID)
	authentication, err := service.Login(context.Background(), LoginInput{Email: email, Password: validTestPassword})
	require.NoError(t, err)

	next := service.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, authentication.User.ID, identity.UserID)
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+authentication.AccessToken)
	response := httptest.NewRecorder()
	next.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestAuthenticateRejectsMissingBearerToken(t *testing.T) {
	service := newTestService(t)
	called := false
	next := service.Authenticate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	response := httptest.NewRecorder()
	next.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.JSONEq(t, `{"error":{"code":"unauthenticated","message":"Authentication required"}}`, response.Body.String())
}
