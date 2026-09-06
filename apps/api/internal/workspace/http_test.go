package workspace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/auth"
)

func TestHandlerRequiresAuthentication(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	authentication := auth.NewService(
		auth.NewRepository(pool),
		auth.NewTokenManager([]byte("01234567890123456789012345678901"), "taskline-api", "taskline-web", nil),
		nil,
	)
	handler := NewHandler(NewService(NewRepository(pool)), authentication)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.JSONEq(t, `{"error":{"code":"unauthenticated","message":"Authentication required"}}`, response.Body.String())
}
