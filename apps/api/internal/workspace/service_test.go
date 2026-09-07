package workspace

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/auth"
)

func TestServiceRejectsDemotingOrRemovingAnOwner(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	owner, err := auth.NewRepository(pool).CreateUser(ctx, auth.CreateUserInput{
		Email:        fmt.Sprintf("service-owner-%d@example.com", time.Now().UnixNano()),
		Name:         "Service owner",
		PasswordHash: "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
	})
	require.NoError(t, err)

	service := NewService(NewRepository(pool))
	workspace, err := service.Create(ctx, owner.ID, "Platform")
	require.NoError(t, err)

	require.ErrorIs(t, service.ChangeMemberRole(ctx, owner.ID, workspace.ID, owner.ID, RoleAdmin), ErrInvalidRequest)
	require.ErrorIs(t, service.RemoveMember(ctx, owner.ID, workspace.ID, owner.ID), ErrInvalidRequest)
}

func TestSlugFromNameUsesOnlyURLSafeASCII(t *testing.T) {
	require.Equal(t, "website-redesign", slugFromName("Website Redesign"))
	require.Empty(t, slugFromName("Проєкт"))
}
