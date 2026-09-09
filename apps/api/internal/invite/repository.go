package invite

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	database "taskline/apps/api/internal/platform/database/sqlc"
	"taskline/apps/api/internal/workspace"
)

var ErrUnavailable = errors.New("invite unavailable")

type Invite struct {
	ID              uuid.UUID      `json:"id"`
	WorkspaceID     uuid.UUID      `json:"workspaceId"`
	Email           string         `json:"email"`
	Role            workspace.Role `json:"role"`
	CreatedByUserID uuid.UUID      `json:"createdByUserId"`
	CreatedByName   string         `json:"createdByName"`
	ExpiresAt       time.Time      `json:"expiresAt"`
	CreatedAt       time.Time      `json:"createdAt"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func hashToken(secret string) [sha256.Size]byte {
	return sha256.Sum256([]byte(secret))
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (r *Repository) Create(ctx context.Context, workspaceID, creatorID uuid.UUID, email string, role workspace.Role, tokenHash [sha256.Size]byte, expiresAt time.Time) (Invite, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Invite{}, err
	}
	defer tx.Rollback(ctx)
	q := database.New(tx)
	normalized := normalizeEmail(email)
	if err := q.RevokeOpenWorkspaceInvitesForEmail(ctx, database.RevokeOpenWorkspaceInvitesForEmailParams{WorkspaceID: workspaceID, Email: normalized}); err != nil {
		return Invite{}, err
	}
	row, err := q.CreateWorkspaceInvite(ctx, database.CreateWorkspaceInviteParams{WorkspaceID: workspaceID, Email: normalized, Role: database.WorkspaceRole(role), TokenHash: tokenHash[:], CreatedByUserID: creatorID, ExpiresAt: expiresAt})
	if err != nil {
		return Invite{}, fmt.Errorf("create invite: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Invite{}, err
	}
	return Invite{ID: row.ID, WorkspaceID: row.WorkspaceID, Email: row.Email, Role: workspace.Role(row.Role), CreatedByUserID: row.CreatedByUserID, ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt}, nil
}

func (r *Repository) Accept(ctx context.Context, tokenHash [sha256.Size]byte, userID uuid.UUID, email string) (workspace.Membership, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return workspace.Membership{}, err
	}
	defer tx.Rollback(ctx)
	q := database.New(tx)
	row, err := q.GetWorkspaceInviteByTokenHashForUpdate(ctx, tokenHash[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return workspace.Membership{}, ErrUnavailable
	}
	if err != nil {
		return workspace.Membership{}, err
	}
	if row.AcceptedAt.Valid || row.RevokedAt.Valid || !row.ExpiresAt.After(time.Now()) || row.Email != normalizeEmail(email) {
		return workspace.Membership{}, ErrUnavailable
	}
	member, err := q.CreateWorkspaceMember(ctx, database.CreateWorkspaceMemberParams{WorkspaceID: row.WorkspaceID, UserID: userID, Role: row.Role, AddedByUserID: pgtype.UUID{Bytes: row.CreatedByUserID, Valid: true}})
	if err != nil {
		return workspace.Membership{}, err
	}
	count, err := q.AcceptWorkspaceInvite(ctx, row.ID)
	if err != nil || count != 1 {
		return workspace.Membership{}, ErrUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return workspace.Membership{}, err
	}
	addedBy := uuid.UUID(member.AddedByUserID.Bytes)
	return workspace.Membership{WorkspaceID: member.WorkspaceID, UserID: member.UserID, Role: workspace.Role(member.Role), AddedByUserID: &addedBy, CreatedAt: member.CreatedAt}, nil
}

func (r *Repository) EmailForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := database.New(r.pool).GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnavailable
	}
	return user.Email, err
}
