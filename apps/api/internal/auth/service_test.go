package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	database "taskline/apps/api/internal/platform/database/sqlc"
)

const validTestPassword = "correct horse battery staple1"

func TestServiceLoginDoesNotRevealWhetherEmailExists(t *testing.T) {
	service := newTestService(t)
	email := fmt.Sprintf("ada-%d@example.com", time.Now().UnixNano())
	_, err := service.Register(context.Background(), RegisterInput{Name: "Ada", Email: email, Password: validTestPassword})
	require.NoError(t, err)

	_, missingErr := service.Login(context.Background(), LoginInput{Email: "missing@example.com", Password: validTestPassword})
	_, wrongPasswordErr := service.Login(context.Background(), LoginInput{Email: email, Password: "a different valid password1"})

	require.ErrorIs(t, missingErr, ErrInvalidCredentials)
	require.ErrorIs(t, wrongPasswordErr, ErrInvalidCredentials)
}

func TestServiceRefreshRotatesAndRejectsReplay(t *testing.T) {
	service := newTestService(t)
	email := fmt.Sprintf("grace-%d@example.com", time.Now().UnixNano())
	registration, err := service.Register(context.Background(), RegisterInput{Name: "Grace", Email: email, Password: validTestPassword})
	require.NoError(t, err)
	verifyTestUser(t, service, registration.User.ID)
	initial, err := service.Login(context.Background(), LoginInput{Email: email, Password: validTestPassword})
	require.NoError(t, err)

	next, err := service.Refresh(context.Background(), initial.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, initial.RefreshToken, next.RefreshToken)

	_, err = service.Refresh(context.Background(), initial.RefreshToken)
	require.ErrorIs(t, err, ErrUnauthenticated)
	_, err = service.Refresh(context.Background(), next.RefreshToken)
	require.ErrorIs(t, err, ErrUnauthenticated)
}

func TestServiceVerificationEnablesLoginAndRejectsTokenReuse(t *testing.T) {
	service := newTestService(t)
	email := fmt.Sprintf("verify-%d@example.com", time.Now().UnixNano())
	registration, err := service.Register(context.Background(), RegisterInput{Name: "Verify", Email: email, Password: validTestPassword})
	require.NoError(t, err)
	_, err = service.Login(context.Background(), LoginInput{Email: email, Password: validTestPassword})
	require.ErrorIs(t, err, ErrEmailVerificationRequired)

	secret := "verification-token-for-test"
	err = service.repository.queries.DeleteOpenEmailVerificationTokens(context.Background(), registration.User.ID)
	require.NoError(t, err)
	hash := HashEmailVerificationToken(secret)
	_, err = service.repository.queries.CreateEmailVerificationToken(context.Background(), database.CreateEmailVerificationTokenParams{UserID: registration.User.ID, TokenHash: hash[:], ExpiresAt: time.Now().Add(time.Hour)})
	require.NoError(t, err)

	require.NoError(t, service.VerifyEmail(context.Background(), secret))
	_, err = service.Login(context.Background(), LoginInput{Email: email, Password: validTestPassword})
	require.NoError(t, err)
	require.ErrorIs(t, service.VerifyEmail(context.Background(), secret), ErrVerificationUnavailable)
}

func verifyTestUser(t *testing.T, service *Service, userID uuid.UUID) {
	t.Helper()
	count, err := service.repository.queries.VerifyUserEmail(context.Background(), userID)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return NewService(NewRepository(pool), NewTokenManager([]byte("01234567890123456789012345678901"), "taskline-api", "taskline-web", time.Now), time.Now)
}
