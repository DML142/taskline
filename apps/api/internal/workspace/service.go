package workspace

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrMemberExists      = errors.New("member exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrSlugExists        = errors.New("workspace slug exists")
)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name string) (WorkspaceSummary, error) {
	if !validName(name) || !validSlug(slugFromName(name)) {
		return WorkspaceSummary{}, ErrInvalidRequest
	}
	workspace, err := s.repository.CreateWithOwner(ctx, userID, name)
	if uniqueViolation(err) {
		return WorkspaceSummary{}, ErrSlugExists
	}
	if err != nil {
		return WorkspaceSummary{}, err
	}
	return WorkspaceSummary{Workspace: workspace, Role: RoleOwner}, nil
}

func validSlug(slug string) bool {
	if len(slug) < 2 || len(slug) > 80 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	for index, character := range slug {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			continue
		}
		if character == '-' && index > 0 && index < len(slug)-1 && slug[index-1] != '-' && slug[index+1] != '-' {
			continue
		}
		return false
	}
	return true
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]WorkspaceSummary, error) {
	return s.repository.ListForUser(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, workspaceID uuid.UUID) (WorkspaceSummary, error) {
	workspace, err := s.repository.GetForMember(ctx, workspaceID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceSummary{}, ErrWorkspaceNotFound
	}
	return workspace, err
}

func (s *Service) Rename(ctx context.Context, actorID, workspaceID uuid.UUID, name string) (Workspace, error) {
	if !validName(name) {
		return Workspace{}, ErrInvalidRequest
	}
	if err := s.requireOwner(ctx, actorID, workspaceID); err != nil {
		return Workspace{}, err
	}
	workspace, err := s.repository.UpdateName(ctx, workspaceID, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return workspace, err
}

func (s *Service) Delete(ctx context.Context, actorID, workspaceID uuid.UUID) error {
	if err := s.requireOwner(ctx, actorID, workspaceID); err != nil {
		return err
	}
	deleted, err := s.repository.Delete(ctx, workspaceID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrWorkspaceNotFound
	}
	return nil
}

func (s *Service) ListMembers(ctx context.Context, actorID, workspaceID uuid.UUID) ([]Member, error) {
	if err := s.requireMember(ctx, actorID, workspaceID); err != nil {
		return nil, err
	}
	return s.repository.ListMembers(ctx, workspaceID)
}

func (s *Service) AddMember(ctx context.Context, actorID, workspaceID uuid.UUID, email string, role Role) (Membership, error) {
	if err := s.requireOwner(ctx, actorID, workspaceID); err != nil {
		return Membership{}, err
	}
	if !manageableRole(role) {
		return Membership{}, ErrInvalidRequest
	}
	userID, err := s.repository.FindUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrUserNotFound
	}
	if err != nil {
		return Membership{}, err
	}
	member, err := s.repository.AddMember(ctx, workspaceID, userID, role)
	if uniqueViolation(err) {
		return Membership{}, ErrMemberExists
	}
	return member, err
}

func (s *Service) ChangeMemberRole(ctx context.Context, actorID, workspaceID, userID uuid.UUID, role Role) error {
	if err := s.requireOwner(ctx, actorID, workspaceID); err != nil {
		return err
	}
	if !manageableRole(role) {
		return ErrInvalidRequest
	}
	target, err := s.repository.GetMember(ctx, workspaceID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkspaceNotFound
	}
	if err != nil {
		return err
	}
	if target.Role == RoleOwner {
		return ErrInvalidRequest
	}
	_, err = s.repository.UpdateMemberRole(ctx, workspaceID, userID, role)
	return err
}

func (s *Service) RemoveMember(ctx context.Context, actorID, workspaceID, userID uuid.UUID) error {
	actor, err := s.repository.GetMember(ctx, workspaceID, actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkspaceNotFound
	}
	if err != nil {
		return err
	}
	target, err := s.repository.GetMember(ctx, workspaceID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkspaceNotFound
	}
	if err != nil {
		return err
	}
	if actorID == userID || target.Role == RoleOwner {
		return ErrInvalidRequest
	}
	if actor.Role != RoleOwner {
		if actor.Role != RoleAdmin {
			return ErrForbidden
		}
		if target.Role == RoleAdmin && (target.AddedByUserID == nil || *target.AddedByUserID != actorID) {
			return ErrForbidden
		}
	}
	deleted, err := s.repository.DeleteMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrWorkspaceNotFound
	}
	return nil
}

func (s *Service) requireMember(ctx context.Context, userID, workspaceID uuid.UUID) error {
	_, err := s.repository.GetMember(ctx, workspaceID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkspaceNotFound
	}
	return err
}

func (s *Service) requireOwner(ctx context.Context, userID, workspaceID uuid.UUID) error {
	member, err := s.repository.GetMember(ctx, workspaceID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkspaceNotFound
	}
	if err != nil {
		return err
	}
	if member.Role != RoleOwner {
		return ErrForbidden
	}
	return nil
}

func validName(name string) bool {
	name = strings.TrimSpace(name)
	return utf8.RuneCountInString(name) >= 1 && utf8.RuneCountInString(name) <= 120
}

func manageableRole(role Role) bool {
	return role == RoleAdmin || role == RoleMember || role == RoleViewer
}

func uniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}
