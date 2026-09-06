package workspace

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/auth"
)

func TestRepositoryCreatesWorkspaceAndOwnerAtomically(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	users := auth.NewRepository(pool)
	owner, err := users.CreateUser(ctx, auth.CreateUserInput{
		Email:        fmt.Sprintf("workspace-owner-%d@example.com", time.Now().UnixNano()),
		Name:         "Workspace owner",
		PasswordHash: "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
	})
	require.NoError(t, err)

	repository := NewRepository(pool)
	created, err := repository.CreateWithOwner(ctx, owner.ID, "Platform")
	require.NoError(t, err)

	membership, err := repository.GetMember(ctx, created.ID, owner.ID)
	require.NoError(t, err)
	require.Equal(t, RoleOwner, membership.Role)
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for repository integration tests")
	}
	return databaseURL
}
