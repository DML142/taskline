package auth

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestRepositoryCreatesAndFindsUserByNormalizedEmail(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	repository := NewRepository(pool)
	email := fmt.Sprintf("ada-%d@example.com", time.Now().UnixNano())
	created, err := repository.CreateUser(ctx, CreateUserInput{
		Email:        email,
		Name:         "Ada Lovelace",
		PasswordHash: "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
	})
	require.NoError(t, err)

	found, err := repository.FindUserByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, email, found.Email)
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for repository integration tests")
	}
	return databaseURL
}
