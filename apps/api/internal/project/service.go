package project

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"taskline/apps/api/internal/workspace"
)

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound       = errors.New("project not found")
	ErrSlugExists     = errors.New("project slug exists")
)

type Project struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type Repository interface {
	Role(context.Context, uuid.UUID, uuid.UUID) (workspace.Role, error)
	Create(context.Context, uuid.UUID, CreateInput) (Project, error)
	List(context.Context, uuid.UUID) ([]Project, error)
	Get(context.Context, uuid.UUID, string) (Project, error)
	Update(context.Context, uuid.UUID, string, CreateInput, bool) (Project, error)
}

func (s *Service) List(ctx context.Context, actorID, workspaceID uuid.UUID) ([]Project, error) {
	if _, err := s.repository.Role(ctx, workspaceID, actorID); err != nil {
		return nil, err
	}
	return s.repository.List(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, actorID, workspaceID uuid.UUID, slug string) (Project, error) {
	if _, err := s.repository.Role(ctx, workspaceID, actorID); err != nil {
		return Project{}, err
	}
	return s.repository.Get(ctx, workspaceID, slug)
}

func (s *Service) Update(ctx context.Context, actorID, workspaceID uuid.UUID, slug string, input CreateInput, archived bool) (Project, error) {
	role, err := s.repository.Role(ctx, workspaceID, actorID)
	if err != nil {
		return Project{}, err
	}
	if role != workspace.RoleOwner && role != workspace.RoleAdmin {
		return Project{}, ErrForbidden
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	if !validName(input.Name) || !validSlug(input.Slug) || len(input.Description) > 2000 {
		return Project{}, ErrInvalidRequest
	}
	project, err := s.repository.Update(ctx, workspaceID, slug, input, archived)
	if uniqueViolation(err) {
		return Project{}, ErrSlugExists
	}
	return project, err
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, actorID, workspaceID uuid.UUID, input CreateInput) (Project, error) {
	role, err := s.repository.Role(ctx, workspaceID, actorID)
	if err != nil {
		return Project{}, err
	}
	if role != workspace.RoleOwner && role != workspace.RoleAdmin {
		return Project{}, ErrForbidden
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Slug == "" {
		input.Slug = slugFromName(input.Name)
	}
	if !validName(input.Name) || !validSlug(input.Slug) || len(input.Description) > 2000 {
		return Project{}, ErrInvalidRequest
	}
	project, err := s.repository.Create(ctx, workspaceID, input)
	if uniqueViolation(err) {
		return Project{}, ErrSlugExists
	}
	return project, err
}

func validName(name string) bool {
	return len([]rune(name)) >= 1 && len([]rune(name)) <= 120
}

func uniqueViolation(err error) bool {
	type postgresError interface{ SQLState() string }
	var databaseError postgresError
	return errors.As(err, &databaseError) && databaseError.SQLState() == "23505"
}

func slugFromName(name string) string {
	var builder strings.Builder
	needsHyphen := false
	for _, character := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if needsHyphen && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(character)
			needsHyphen = false
		} else {
			needsHyphen = true
		}
	}
	return builder.String()
}

func validSlug(slug string) bool {
	if len(slug) < 2 || len(slug) > 80 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	previousHyphen := false
	for _, character := range slug {
		if character == '-' {
			if previousHyphen {
				return false
			}
			previousHyphen = true
			continue
		}
		if !((character >= 'a' && character <= 'z') || (character >= '0' && character <= '9')) {
			return false
		}
		previousHyphen = false
	}
	return true
}
