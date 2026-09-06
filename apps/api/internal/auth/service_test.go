package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const validTestPassword = "correct horse battery staple"

func TestServiceLoginDoesNotRevealWhetherEmailExists(t *testing.T) {
	service := newTestService(t)
	email := fmt.Sprintf("ada-%d@example.com", time.Now().UnixNano())
	_, err := service.Register(context.Background(), RegisterInput{Name: "Ada", Email: email, Password: validTestPassword})
	require.NoError(t, err)

	_, missingErr := service.Login(context.Background(), LoginInput{Email: "missing@example.com", Password: validTestPassword})
	_, wrongPasswordErr := service.Login(context.Background(), LoginInput{Email: email, Password: "a different valid password"})

	require.ErrorIs(t, missingErr, ErrInvalidCredentials)
	require.ErrorIs(t, wrongPasswordErr, ErrInvalidCredentials)
}

func TestServiceRefreshRotatesAndRejectsReplay(t *testing.T) {
	service := newTestService(t)
	email := fmt.Sprintf("grace-%d@example.com", time.Now().UnixNano())
	initial, err := service.Register(context.Background(), RegisterInput{Name: "Grace", Email: email, Password: validTestPassword})
	require.NoError(t, err)

	next, err := service.Refresh(context.Background(), initial.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, initial.RefreshToken, next.RefreshToken)

	_, err = service.Refresh(context.Background(), initial.RefreshToken)
	require.ErrorIs(t, err, ErrUnauthenticated)
	_, err = service.Refresh(context.Background(), next.RefreshToken)
	require.ErrorIs(t, err, ErrUnauthenticated)
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return NewService(NewRepository(pool), NewTokenManager([]byte("01234567890123456789012345678901"), "taskline-api", "taskline-web", time.Now), time.Now)
}
