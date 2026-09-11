package issue

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *PostgresRepository) Project(ctx context.Context, workspaceID uuid.UUID, slug string) (Project, error) {
	project, err := database.New(r.pool).GetProjectBySlug(ctx, database.GetProjectBySlugParams{WorkspaceID: workspaceID, Slug: slug})
	return Project{ID: project.ID}, err
}

func (r *PostgresRepository) Member(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	_, err := database.New(r.pool).GetWorkspaceMember(ctx, database.GetWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *PostgresRepository) Create(ctx context.Context, projectID, creatorID uuid.UUID, input CreateInput) (Issue, error) {
	row, err := database.New(r.pool).CreateIssue(ctx, database.CreateIssueParams{
		ProjectID: projectID, Title: input.Title, Description: input.Description, Status: string(input.Status), Priority: string(input.Priority), CreatorID: creatorID, AssigneeID: nullableUUID(input.AssigneeID),
	})
	return issueFromRow(row), err
}

func (r *PostgresRepository) List(ctx context.Context, projectID uuid.UUID, filter ListFilter) ([]Issue, error) {
	rows, err := database.New(r.pool).ListIssuesForProjectByFilters(ctx, database.ListIssuesForProjectByFiltersParams{
		ProjectID:  projectID,
		Status:     string(filter.Status),
		AssigneeID: nullableUUID(filter.AssigneeID),
		Priority:   string(filter.Priority),
	})
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, len(rows))
	for index, row := range rows {
		issues[index] = issueFromRow(row)
	}
	return issues, nil
}

func (r *PostgresRepository) Get(ctx context.Context, projectID, issueID uuid.UUID) (Issue, error) {
	row, err := database.New(r.pool).GetIssue(ctx, database.GetIssueParams{ProjectID: projectID, ID: issueID})
	return issueFromRow(row), err
}

func (r *PostgresRepository) Update(ctx context.Context, projectID, issueID uuid.UUID, input UpdateInput) (Issue, error) {
	row, err := database.New(r.pool).UpdateIssue(ctx, database.UpdateIssueParams{
		ProjectID: projectID, ID: issueID, Title: input.Title, Description: input.Description, Status: string(input.Status), Priority: string(input.Priority), AssigneeID: nullableUUID(input.AssigneeID),
	})
	return issueFromRow(row), err
}

func nullableUUID(value *uuid.UUID) pgtype.UUID {
	if value == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *value, Valid: true}
}

func issueFromRow(row database.Issue) Issue {
	issue := Issue{ID: row.ID, ProjectID: row.ProjectID, Title: row.Title, Description: row.Description, Status: Status(row.Status), Priority: Priority(row.Priority), CreatorID: row.CreatorID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.AssigneeID.Valid {
		assigneeID := uuid.UUID(row.AssigneeID.Bytes)
		issue.AssigneeID = &assigneeID
	}
	return issue
}
