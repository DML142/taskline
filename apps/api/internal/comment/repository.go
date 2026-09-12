package comment

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

func (r *PostgresRepository) Project(ctx context.Context, workspaceID uuid.UUID, slug string) (Project, error) {
	project, err := database.New(r.pool).GetProjectBySlug(ctx, database.GetProjectBySlugParams{WorkspaceID: workspaceID, Slug: slug})
	return Project{ID: project.ID}, err
}

func (r *PostgresRepository) Issue(ctx context.Context, projectID, issueID uuid.UUID) error {
	_, err := database.New(r.pool).GetIssue(ctx, database.GetIssueParams{ProjectID: projectID, ID: issueID})
	return err
}

func (r *PostgresRepository) Create(ctx context.Context, issueID, authorID uuid.UUID, input CreateInput) (Comment, error) {
	queries := database.New(r.pool)
	commentID, err := queries.CreateIssueComment(ctx, database.CreateIssueCommentParams{IssueID: issueID, AuthorID: authorID, Body: input.Body})
	if err != nil {
		return Comment{}, err
	}
	return r.Get(ctx, issueID, commentID)
}

func (r *PostgresRepository) Get(ctx context.Context, issueID, commentID uuid.UUID) (Comment, error) {
	row, err := database.New(r.pool).GetIssueComment(ctx, database.GetIssueCommentParams{IssueID: issueID, ID: commentID})
	return commentFromRow(row), err
}

func (r *PostgresRepository) Update(ctx context.Context, issueID, commentID uuid.UUID, input UpdateInput) (Comment, error) {
	queries := database.New(r.pool)
	if err := queries.UpdateIssueComment(ctx, database.UpdateIssueCommentParams{IssueID: issueID, ID: commentID, Body: input.Body}); err != nil {
		return Comment{}, err
	}
	return r.Get(ctx, issueID, commentID)
}

func (r *PostgresRepository) Delete(ctx context.Context, issueID, commentID uuid.UUID) error {
	return database.New(r.pool).DeleteIssueComment(ctx, database.DeleteIssueCommentParams{IssueID: issueID, ID: commentID})
}

func (r *PostgresRepository) List(ctx context.Context, issueID uuid.UUID) ([]Comment, error) {
	rows, err := database.New(r.pool).ListIssueComments(ctx, issueID)
	if err != nil {
		return nil, err
	}
	comments := make([]Comment, len(rows))
	for index, row := range rows {
		comments[index] = commentFromListRow(row)
	}
	return comments, nil
}

func commentFromRow(row database.GetIssueCommentRow) Comment {
	return Comment{ID: row.ID, IssueID: row.IssueID, AuthorID: row.AuthorID, AuthorName: row.AuthorName, Body: row.Body, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func commentFromListRow(row database.ListIssueCommentsRow) Comment {
	return Comment{ID: row.ID, IssueID: row.IssueID, AuthorID: row.AuthorID, AuthorName: row.AuthorName, Body: row.Body, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
