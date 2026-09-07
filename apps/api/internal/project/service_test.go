package project

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/workspace"
)

func TestSlugFromName(t *testing.T) {
	if got, want := slugFromName("  Website Redesign  "), "website-redesign"; got != want {
		t.Fatalf("slugFromName() = %q, want %q", got, want)
	}
}

func TestValidSlug(t *testing.T) {
	for _, slug := range []string{"website-redesign", "api-v2", "a1"} {
		if !validSlug(slug) {
			t.Errorf("validSlug(%q) = false, want true", slug)
		}
	}
	for _, slug := range []string{"Website", "two--hyphens", "-leading", "trailing-", ""} {
		if validSlug(slug) {
			t.Errorf("validSlug(%q) = true, want false", slug)
		}
	}
}

func TestCreateAllowsAdminAndGeneratesSlug(t *testing.T) {
	repository := &fakeRepository{role: workspace.RoleAdmin}
	service := NewService(repository)

	created, err := service.Create(context.Background(), uuid.New(), uuid.New(), CreateInput{
		Name:        "Website Redesign",
		Description: "Refresh the marketing site",
	})

	require.NoError(t, err)
	require.Equal(t, "website-redesign", created.Slug)
	require.Equal(t, "website-redesign", repository.input.Slug)
}

func TestCreateRejectsMember(t *testing.T) {
	service := NewService(&fakeRepository{role: workspace.RoleMember})

	_, err := service.Create(context.Background(), uuid.New(), uuid.New(), CreateInput{Name: "Website"})

	require.ErrorIs(t, err, ErrForbidden)
}

type fakeRepository struct {
	role  workspace.Role
	input CreateInput
}

func (r *fakeRepository) Role(context.Context, uuid.UUID, uuid.UUID) (workspace.Role, error) {
	return r.role, nil
}

func (r *fakeRepository) Create(_ context.Context, _ uuid.UUID, input CreateInput) (Project, error) {
	r.input = input
	return Project{ID: uuid.New(), Name: input.Name, Slug: input.Slug, Description: input.Description}, nil
}

func (r *fakeRepository) List(context.Context, uuid.UUID) ([]Project, error) { return nil, nil }

func (r *fakeRepository) Get(context.Context, uuid.UUID, string) (Project, error) {
	return Project{}, nil
}

func (r *fakeRepository) Update(context.Context, uuid.UUID, string, CreateInput, bool) (Project, error) {
	return Project{}, nil
}
