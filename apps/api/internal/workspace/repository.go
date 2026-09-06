package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	database "taskline/apps/api/internal/platform/database/sqlc"
)

type Role string

const (
	RoleOwner  Role = "OWNER"
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
	RoleViewer Role = "VIEWER"
)

type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Membership struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	UserID      uuid.UUID `json:"userId"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"createdAt"`
}

type WorkspaceSummary struct {
	Workspace
	Role Role `json:"role"`
}

type Member struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) CreateWithOwner(ctx context.Context, ownerID uuid.UUID, name string) (Workspace, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Workspace{}, fmt.Errorf("begin workspace transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := database.New(tx)
	workspace, err := queries.CreateWorkspace(ctx, strings.TrimSpace(name))
	if err != nil {
		return Workspace{}, fmt.Errorf("create workspace: %w", err)
	}
	if _, err := queries.CreateWorkspaceMember(ctx, database.CreateWorkspaceMemberParams{
		WorkspaceID: workspace.ID,
		UserID:      ownerID,
		Role:        database.WorkspaceRole(RoleOwner),
	}); err != nil {
		return Workspace{}, fmt.Errorf("create workspace owner: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Workspace{}, fmt.Errorf("commit workspace transaction: %w", err)
	}
	return Workspace{ID: workspace.ID, Name: workspace.Name, CreatedAt: workspace.CreatedAt, UpdatedAt: workspace.UpdatedAt}, nil
}

func (r *Repository) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (Membership, error) {
	member, err := database.New(r.pool).GetWorkspaceMember(ctx, database.GetWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return Membership{}, fmt.Errorf("get workspace member: %w", err)
	}
	return Membership{WorkspaceID: member.WorkspaceID, UserID: member.UserID, Role: Role(member.Role), CreatedAt: member.CreatedAt}, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceSummary, error) {
	rows, err := database.New(r.pool).ListWorkspacesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	result := make([]WorkspaceSummary, len(rows))
	for i, row := range rows {
		result[i] = WorkspaceSummary{Workspace: Workspace{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, Role: Role(row.Role)}
	}
	return result, nil
}

func (r *Repository) GetForMember(ctx context.Context, workspaceID, userID uuid.UUID) (WorkspaceSummary, error) {
	row, err := database.New(r.pool).GetWorkspaceForMember(ctx, database.GetWorkspaceForMemberParams{ID: workspaceID, UserID: userID})
	if err != nil {
		return WorkspaceSummary{}, fmt.Errorf("get workspace: %w", err)
	}
	return WorkspaceSummary{Workspace: Workspace{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, Role: Role(row.Role)}, nil
}

func (r *Repository) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error) {
	rows, err := database.New(r.pool).ListWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	result := make([]Member, len(rows))
	for i, row := range rows {
		result[i] = Member{UserID: row.ID, Email: row.Email, Name: row.Name, Role: Role(row.Role), CreatedAt: row.CreatedAt}
	}
	return result, nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	user, err := database.New(r.pool).FindWorkspaceUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return uuid.Nil, fmt.Errorf("find workspace user: %w", err)
	}
	return user.ID, nil
}

func (r *Repository) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role Role) (Membership, error) {
	member, err := database.New(r.pool).CreateWorkspaceMember(ctx, database.CreateWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID, Role: database.WorkspaceRole(role)})
	if err != nil {
		return Membership{}, fmt.Errorf("add workspace member: %w", err)
	}
	return Membership{WorkspaceID: member.WorkspaceID, UserID: member.UserID, Role: Role(member.Role), CreatedAt: member.CreatedAt}, nil
}

func (r *Repository) UpdateName(ctx context.Context, workspaceID uuid.UUID, name string) (Workspace, error) {
	row, err := database.New(r.pool).UpdateWorkspaceName(ctx, database.UpdateWorkspaceNameParams{ID: workspaceID, Name: strings.TrimSpace(name)})
	if err != nil {
		return Workspace{}, fmt.Errorf("update workspace name: %w", err)
	}
	return Workspace{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role Role) (Membership, error) {
	member, err := database.New(r.pool).UpdateWorkspaceMemberRole(ctx, database.UpdateWorkspaceMemberRoleParams{WorkspaceID: workspaceID, UserID: userID, Role: database.WorkspaceRole(role)})
	if err != nil {
		return Membership{}, fmt.Errorf("update workspace member role: %w", err)
	}
	return Membership{WorkspaceID: member.WorkspaceID, UserID: member.UserID, Role: Role(member.Role), CreatedAt: member.CreatedAt}, nil
}

func (r *Repository) DeleteMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	count, err := database.New(r.pool).DeleteWorkspaceMember(ctx, database.DeleteWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("delete workspace member: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) Delete(ctx context.Context, workspaceID uuid.UUID) (bool, error) {
	count, err := database.New(r.pool).DeleteWorkspace(ctx, workspaceID)
	if err != nil {
		return false, fmt.Errorf("delete workspace: %w", err)
	}
	return count > 0, nil
}
