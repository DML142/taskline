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
	"taskline/apps/api/internal/mailer"
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
	mailer     mailer.Mailer
	now        func() time.Time
	webOrigin  string
}

func NewService(repository *Repository, workspaces *workspace.Service, delivery mailer.Mailer, webOrigin string, now func() time.Time) *Service {
	if delivery == nil {
		delivery = mailer.Disabled{}
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, workspaces: workspaces, mailer: delivery, webOrigin: strings.TrimRight(webOrigin, "/"), now: now}
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
	if err := s.mailer.Send(ctx, mailer.Message{To: invite.Email, Subject: "Invitation to " + current.Name, Body: "Accept this workspace invitation: " + s.webOrigin + "/invites/" + secret}); err != nil {
		_, _ = s.repository.Revoke(ctx, workspaceID, invite.ID, nil)
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

func (s *Service) Preview(ctx context.Context, rawToken string) (Preview, error) {
	if rawToken == "" {
		return Preview{}, ErrUnavailable
	}
	return s.repository.Preview(ctx, hashToken(rawToken))
}

func (s *Service) List(ctx context.Context, actorID, workspaceID uuid.UUID) ([]Invite, error) {
	current, err := s.workspaces.Get(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if current.Role == workspace.RoleOwner {
		return s.repository.List(ctx, workspaceID, nil)
	}
	if current.Role == workspace.RoleAdmin {
		return s.repository.List(ctx, workspaceID, &actorID)
	}
	return nil, ErrForbidden
}

func (s *Service) Revoke(ctx context.Context, actorID, workspaceID, inviteID uuid.UUID) error {
	current, err := s.workspaces.Get(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}
	var creatorID *uuid.UUID
	if current.Role == workspace.RoleAdmin {
		creatorID = &actorID
	} else if current.Role != workspace.RoleOwner {
		return ErrForbidden
	}
	revoked, err := s.repository.Revoke(ctx, workspaceID, inviteID, creatorID)
	if err != nil {
		return err
	}
	if !revoked {
		return ErrUnavailable
	}
	return nil
}
