package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	database "taskline/apps/api/internal/platform/database/sqlc"
)

const refreshSessionLifetime = 30 * 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

type RegisterInput struct{ Name, Email, Password string }
type LoginInput struct{ Email, Password string }
type Authentication struct {
	User                      StoredUser
	AccessToken, RefreshToken string
}
type Service struct {
	repository *Repository
	tokens     *TokenManager
	now        func() time.Time
}

func NewService(repository *Repository, tokens *TokenManager, now func() time.Time) *Service {
	return &Service{repository: repository, tokens: tokens, now: now}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (Authentication, error) {
	hash, err := HashPassword(input.Password)
	if err != nil {
		return Authentication{}, err
	}
	return s.repository.withTx(ctx, func(q *database.Queries) (Authentication, error) {
		user, err := q.CreateUser(ctx, database.CreateUserParams{Email: strings.ToLower(strings.TrimSpace(input.Email)), Name: strings.TrimSpace(input.Name), PasswordHash: hash})
		if err != nil {
			return Authentication{}, err
		}
		return s.newAuthentication(ctx, q, storedUser(user), uuid.New())
	})
}

func (s *Service) Login(ctx context.Context, input LoginInput) (Authentication, error) {
	user, err := s.repository.FindUserByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || ComparePassword(user.PasswordHash, input.Password) != nil {
		return Authentication{}, ErrInvalidCredentials
	}
	return s.repository.withTx(ctx, func(q *database.Queries) (Authentication, error) {
		return s.newAuthentication(ctx, q, user, uuid.New())
	})
}

func (s *Service) Refresh(ctx context.Context, token string) (Authentication, error) {
	replayed := false
	authentication, err := s.repository.withTx(ctx, func(q *database.Queries) (Authentication, error) {
		session, err := q.GetSessionByTokenHashForUpdate(ctx, HashRefreshToken(token))
		if err != nil {
			return Authentication{}, ErrUnauthenticated
		}
		if session.RevokedAt.Valid || !session.ExpiresAt.After(s.now()) {
			if err := q.RevokeSessionFamily(ctx, session.FamilyID); err != nil {
				return Authentication{}, err
			}
			replayed = true
			return Authentication{}, nil
		}
		user, err := q.GetUserForSession(ctx, session.ID)
		if err != nil {
			return Authentication{}, ErrUnauthenticated
		}
		nextID := uuid.New()
		refreshToken, err := NewRefreshToken()
		if err != nil {
			return Authentication{}, err
		}
		_, err = q.CreateSession(ctx, database.CreateSessionParams{ID: nextID, UserID: user.ID, FamilyID: session.FamilyID, TokenHash: HashRefreshToken(refreshToken), ExpiresAt: s.now().Add(refreshSessionLifetime)})
		if err != nil {
			return Authentication{}, err
		}
		err = q.RevokeSession(ctx, database.RevokeSessionParams{ID: session.ID, ReplacedBy: pgtype.UUID{Bytes: nextID, Valid: true}})
		if err != nil {
			return Authentication{}, err
		}
		accessToken, err := s.tokens.NewAccessToken(user.ID, nextID)
		if err != nil {
			return Authentication{}, err
		}
		authn := Authentication{User: storedUser(user), AccessToken: accessToken, RefreshToken: refreshToken}
		return authn, nil
	})
	if err != nil {
		return Authentication{}, err
	}
	if replayed {
		return Authentication{}, ErrUnauthenticated
	}
	return authentication, nil
}

func (s *Service) CurrentUser(ctx context.Context, accessToken string) (StoredUser, error) {
	identity, err := s.authenticate(ctx, accessToken)
	if err != nil {
		return StoredUser{}, ErrUnauthenticated
	}
	user, err := s.repository.FindUserByID(ctx, identity.UserID)
	if err != nil {
		return StoredUser{}, ErrUnauthenticated
	}
	return user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.repository.queries.RevokeSessionByTokenHash(ctx, HashRefreshToken(refreshToken))
}

func (s *Service) newAuthentication(ctx context.Context, q *database.Queries, user StoredUser, familyID uuid.UUID) (Authentication, error) {
	refreshToken, err := NewRefreshToken()
	if err != nil {
		return Authentication{}, err
	}
	sessionID := uuid.New()
	_, err = q.CreateSession(ctx, database.CreateSessionParams{ID: sessionID, UserID: user.ID, FamilyID: familyID, TokenHash: HashRefreshToken(refreshToken), ExpiresAt: s.now().Add(refreshSessionLifetime)})
	if err != nil {
		return Authentication{}, err
	}
	accessToken, err := s.tokens.NewAccessToken(user.ID, sessionID)
	if err != nil {
		return Authentication{}, err
	}
	return Authentication{User: user, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (r *Repository) withTx(ctx context.Context, fn func(*database.Queries) (Authentication, error)) (Authentication, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Authentication{}, err
	}
	defer tx.Rollback(ctx)
	result, err := fn(database.New(tx))
	if err != nil {
		return Authentication{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Authentication{}, err
	}
	return result, nil
}
