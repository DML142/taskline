package invite

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"taskline/apps/api/internal/workspace"
)

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidRequest = errors.New("invalid request")
	ErrDeliveryFailed = errors.New("invite delivery failed")
)

type Service struct {
	repository *Repository
	workspaces *workspace.Service
	mailer     Mailer
	now        func() time.Time
	webOrigin  string
}

func NewService(repository *Repository, workspaces *workspace.Service, mailer Mailer, webOrigin string, now func() time.Time) *Service {
	if mailer == nil {
		mailer = disabledMailer{}
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, workspaces: workspaces, mailer: mailer, webOrigin: strings.TrimRight(webOrigin, "/"), now: now}
}

func (s *Service) Create(ctx context.Context, actorID, workspaceID uuid.UUID, email string, role workspace.Role) (Invite, error) {
	current, err := s.workspaces.Get(ctx, actorID, workspaceID)
	if err != nil {
		return Invite{}, err
	}
	if current.Role != workspace.RoleOwner && current.Role != workspace.RoleAdmin {
		return Invite{}, ErrForbidden
	}
	if normalizeEmail(email) == "" || (role != workspace.RoleAdmin && role != workspace.RoleMember && role != workspace.RoleViewer) {
		return Invite{}, ErrInvalidRequest
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return Invite{}, err
	}
	secret := base64.RawURLEncoding.EncodeToString(bytes)
	invite, err := s.repository.Create(ctx, workspaceID, actorID, email, role, hashToken(secret), s.now().Add(24*time.Hour))
	if err != nil {
		return Invite{}, err
	}
	if err := s.mailer.Send(ctx, Message{To: invite.Email, WorkspaceName: current.Name, AcceptanceURL: s.webOrigin + "/invites/" + secret}); err != nil {
		return Invite{}, fmt.Errorf("%w: %v", ErrDeliveryFailed, err)
	}
	return invite, nil
}

func (s *Service) Accept(ctx context.Context, rawToken string, userID uuid.UUID, email string) (workspace.Membership, error) {
	if rawToken == "" {
		return workspace.Membership{}, ErrUnavailable
	}
	return s.repository.Accept(ctx, hashToken(rawToken), userID, email)
}

func (s *Service) AcceptForUser(ctx context.Context, rawToken string, userID uuid.UUID) (workspace.Membership, error) {
	email, err := s.repository.EmailForUser(ctx, userID)
	if err != nil {
		return workspace.Membership{}, err
	}
	return s.Accept(ctx, rawToken, userID, email)
}

func unavailable(err error) bool {
	return errors.Is(err, ErrUnavailable) || errors.Is(err, pgx.ErrNoRows)
}
