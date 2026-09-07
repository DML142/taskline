package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	database "taskline/apps/api/internal/platform/database/sqlc"
	"taskline/apps/api/internal/workspace"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{pool: pool} }

func (r *PostgresRepository) Role(ctx context.Context, workspaceID, userID uuid.UUID) (workspace.Role, error) {
	member, err := database.New(r.pool).GetWorkspaceMember(ctx, database.GetWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return "", fmt.Errorf("get workspace member: %w", err)
	}
	return workspace.Role(member.Role), nil
}

func (r *PostgresRepository) Create(ctx context.Context, workspaceID uuid.UUID, input CreateInput) (Project, error) {
	row, err := database.New(r.pool).CreateProject(ctx, database.CreateProjectParams{WorkspaceID: workspaceID, Name: input.Name, Slug: input.Slug, Description: input.Description})
	return projectFromRow(row), err
}

func (r *PostgresRepository) List(ctx context.Context, workspaceID uuid.UUID) ([]Project, error) {
	rows, err := database.New(r.pool).ListProjectsForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	projects := make([]Project, len(rows))
	for index, row := range rows {
		projects[index] = projectFromRow(row)
	}
	return projects, nil
}

func (r *PostgresRepository) Get(ctx context.Context, workspaceID uuid.UUID, slug string) (Project, error) {
	row, err := database.New(r.pool).GetProjectBySlug(ctx, database.GetProjectBySlugParams{WorkspaceID: workspaceID, Slug: slug})
	return projectFromRow(row), err
}

func (r *PostgresRepository) Update(ctx context.Context, workspaceID uuid.UUID, slug string, input CreateInput, archived bool) (Project, error) {
	row, err := database.New(r.pool).UpdateProject(ctx, database.UpdateProjectParams{WorkspaceID: workspaceID, Slug: slug, Name: input.Name, Slug_2: input.Slug, Description: input.Description, Archived: archived})
	return projectFromRow(row), err
}

func projectFromRow(row database.Project) Project {
	return Project{ID: row.ID, WorkspaceID: row.WorkspaceID, Name: row.Name, Slug: row.Slug, Description: row.Description, Archived: row.Archived, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
