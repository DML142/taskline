package issue

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
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrInvalidAssignee = errors.New("invalid assignee")
)

type Status string

const (
	StatusTodo       Status = "TODO"
	StatusInProgress Status = "IN_PROGRESS"
	StatusDone       Status = "DONE"
)

type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
)

type Project struct{ ID uuid.UUID }

type Issue struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"projectId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	CreatorID   uuid.UUID  `json:"creatorId"`
	AssigneeID  *uuid.UUID `json:"assigneeId"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type CreateInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assigneeId"`
}

type UpdateInput CreateInput

type ListFilter struct {
	Status     Status
	AssigneeID *uuid.UUID
}

type Repository interface {
	Role(context.Context, uuid.UUID, uuid.UUID) (workspace.Role, error)
	Project(context.Context, uuid.UUID, string) (Project, error)
	Member(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	Create(context.Context, uuid.UUID, uuid.UUID, CreateInput) (Issue, error)
	List(context.Context, uuid.UUID, ListFilter) ([]Issue, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (Issue, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateInput) (Issue, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, filter ListFilter) ([]Issue, error) {
	if _, err := s.repository.Role(ctx, workspaceID, actorID); err != nil {
		return nil, err
	}
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, ErrInvalidRequest
	}
	if err := s.validAssignee(ctx, workspaceID, filter.AssigneeID); err != nil {
		return nil, err
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return nil, err
	}
	return s.repository.List(ctx, project.ID, filter)
}

func (s *Service) Get(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID uuid.UUID) (Issue, error) {
	if _, err := s.repository.Role(ctx, workspaceID, actorID); err != nil {
		return Issue{}, err
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return Issue{}, err
	}
	return s.repository.Get(ctx, project.ID, issueID)
}

func (s *Service) Create(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, input CreateInput) (Issue, error) {
	role, err := s.repository.Role(ctx, workspaceID, actorID)
	if err != nil {
		return Issue{}, err
	}
	if role == workspace.RoleViewer {
		return Issue{}, ErrForbidden
	}
	input = normalized(input)
	if !validInput(input) {
		return Issue{}, ErrInvalidRequest
	}
	if role == workspace.RoleMember && input.AssigneeID != nil && *input.AssigneeID != actorID {
		return Issue{}, ErrForbidden
	}
	if err := s.validAssignee(ctx, workspaceID, input.AssigneeID); err != nil {
		return Issue{}, err
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return Issue{}, err
	}
	return s.repository.Create(ctx, project.ID, actorID, input)
}

func (s *Service) Update(ctx context.Context, actorID, workspaceID uuid.UUID, projectSlug string, issueID uuid.UUID, input UpdateInput) (Issue, error) {
	role, err := s.repository.Role(ctx, workspaceID, actorID)
	if err != nil {
		return Issue{}, err
	}
	if role == workspace.RoleViewer {
		return Issue{}, ErrForbidden
	}
	project, err := s.repository.Project(ctx, workspaceID, projectSlug)
	if err != nil {
		return Issue{}, err
	}
	current, err := s.repository.Get(ctx, project.ID, issueID)
	if err != nil {
		return Issue{}, err
	}
	if role == workspace.RoleMember && (current.AssigneeID == nil || *current.AssigneeID != actorID || input.AssigneeID == nil || *input.AssigneeID != actorID) {
		return Issue{}, ErrForbidden
	}
	createInput := normalized(CreateInput(input))
	input = UpdateInput(createInput)
	if !validInput(CreateInput(input)) {
		return Issue{}, ErrInvalidRequest
	}
	if err := s.validAssignee(ctx, workspaceID, input.AssigneeID); err != nil {
		return Issue{}, err
	}
	return s.repository.Update(ctx, project.ID, issueID, input)
}

func (s *Service) validAssignee(ctx context.Context, workspaceID uuid.UUID, assigneeID *uuid.UUID) error {
	if assigneeID == nil {
		return nil
	}
	isMember, err := s.repository.Member(ctx, workspaceID, *assigneeID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrInvalidAssignee
	}
	return nil
}

func normalized(input CreateInput) CreateInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Status == "" {
		input.Status = StatusTodo
	}
	if input.Priority == "" {
		input.Priority = PriorityMedium
	}
	return input
}

func validInput(input CreateInput) bool {
	return utf8.RuneCountInString(input.Title) >= 1 && utf8.RuneCountInString(input.Title) <= 200 &&
		utf8.RuneCountInString(input.Description) <= 10000 && validStatus(input.Status) && validPriority(input.Priority)
}

func validStatus(status Status) bool {
	return status == StatusTodo || status == StatusInProgress || status == StatusDone
}

func validPriority(priority Priority) bool {
	return priority == PriorityLow || priority == PriorityMedium || priority == PriorityHigh
}
