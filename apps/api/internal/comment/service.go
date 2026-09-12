package comment

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"taskline/apps/api/internal/workspace"
)

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidRequest = errors.New("invalid request")
)

type Project struct{ ID uuid.UUID }

type Comment struct {
	ID         uuid.UUID `json:"id"`
	IssueID    uuid.UUID `json:"issueId"`
	AuthorID   uuid.UUID `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Body string `json:"body"`
}

type UpdateInput CreateInput

type Repository interface {
	Role(context.Context, uuid.UUID, uuid.UUID) (workspace.Role, error)
	Project(context.Context, uuid.UUID, string) (Project, error)
	Issue(context.Context, uuid.UUID, uuid.UUID) error
	Create(context.Context, uuid.UUID, uuid.UUID, CreateInput) (Comment, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (Comment, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateInput) (Comment, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	List(context.Context, uuid.UUID) ([]Comment, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID uuid.UUID, input CreateInput) (Comment, error) {
	role, err := s.repository.Role(ctx, workspaceID, actorID)
	if err != nil {
		return Comment{}, err
	}
	if role == workspace.RoleViewer {
		return Comment{}, ErrForbidden
	}
	input = normalized(input)
	if !validInput(input) {
		return Comment{}, ErrInvalidRequest
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return Comment{}, err
	}
	if err := s.repository.Issue(ctx, project.ID, issueID); err != nil {
		return Comment{}, err
	}
	return s.repository.Create(ctx, issueID, actorID, input)
}

func (s *Service) List(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID uuid.UUID) ([]Comment, error) {
	if _, err := s.repository.Role(ctx, workspaceID, actorID); err != nil {
		return nil, err
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return nil, err
	}
	if err := s.repository.Issue(ctx, project.ID, issueID); err != nil {
		return nil, err
	}
	return s.repository.List(ctx, issueID)
}

func (s *Service) Delete(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID, commentID uuid.UUID) error {
	if err := s.canModify(ctx, actorID, workspaceID, projectSlug, issueID, commentID); err != nil {
		return err
	}
	return s.repository.Delete(ctx, issueID, commentID)
}

func (s *Service) Update(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID, commentID uuid.UUID, input UpdateInput) (Comment, error) {
	input = UpdateInput(normalized(CreateInput(input)))
	if !validInput(CreateInput(input)) {
		return Comment{}, ErrInvalidRequest
	}
	if err := s.canModify(ctx, actorID, workspaceID, projectSlug, issueID, commentID); err != nil {
		return Comment{}, err
	}
	return s.repository.Update(ctx, issueID, commentID, input)
}

func (s *Service) canModify(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID, commentID uuid.UUID) error {
	role, err := s.repository.Role(ctx, workspaceID, actorID)
	if err != nil {
		return err
	}
	if role == workspace.RoleViewer {
		return ErrForbidden
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return err
	}
	if err := s.repository.Issue(ctx, project.ID, issueID); err != nil {
		return err
	}
	comment, err := s.repository.Get(ctx, issueID, commentID)
	if err != nil {
		return err
	}
	if comment.AuthorID != actorID {
		return ErrForbidden
	}
	return nil
}

func normalized(input CreateInput) CreateInput {
	input.Body = strings.TrimSpace(input.Body)
	return input
}

func validInput(input CreateInput) bool {
	return utf8.RuneCountInString(input.Body) >= 1 && utf8.RuneCountInString(input.Body) <= 10000
}
