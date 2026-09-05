package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	database "taskline/apps/api/internal/platform/database/sqlc"
)

type Repository struct {
	queries *database.Queries
}

type CreateUserInput struct {
	Email        string
	Name         string
	PasswordHash string
}

type StoredUser struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{queries: database.New(pool)}
}

func (r *Repository) CreateUser(ctx context.Context, input CreateUserInput) (StoredUser, error) {
	user, err := r.queries.CreateUser(ctx, database.CreateUserParams{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: input.PasswordHash,
	})
	if err != nil {
		return StoredUser{}, fmt.Errorf("create user: %w", err)
	}
	return storedUser(user), nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return StoredUser{}, fmt.Errorf("find user by email: %w", err)
	}
	return storedUser(user), nil
}

func storedUser(user database.User) StoredUser {
	return StoredUser{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}
