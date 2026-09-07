package issue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/workspace"
)

func TestCreateAllowsMemberToCreateIssue(t *testing.T) {
	repository := &fakeRepository{role: workspace.RoleMember}
	service := NewService(repository)
	workspaceID, actorID := uuid.New(), uuid.New()

	created, err := service.Create(context.Background(), actorID, workspaceID, "website-redesign", CreateInput{
		Title:    "Add issue board",
		Status:   StatusTodo,
		Priority: PriorityHigh,
	})

	require.NoError(t, err)
	require.Equal(t, "Add issue board", created.Title)
	require.Equal(t, actorID, repository.creatorID)
}

func TestCreateRejectsViewer(t *testing.T) {
	service := NewService(&fakeRepository{role: workspace.RoleViewer})

	_, err := service.Create(context.Background(), uuid.New(), uuid.New(), "website-redesign", CreateInput{Title: "Blocked"})

	require.ErrorIs(t, err, ErrForbidden)
}

func TestCreateRejectsAssigneeOutsideWorkspace(t *testing.T) {
	repository := &fakeRepository{role: workspace.RoleAdmin, member: false}
	service := NewService(repository)
	assigneeID := uuid.New()

	_, err := service.Create(context.Background(), uuid.New(), uuid.New(), "website-redesign", CreateInput{Title: "Assign", AssigneeID: &assigneeID})

	require.ErrorIs(t, err, ErrInvalidAssignee)
}

func TestListRejectsAssigneeOutsideWorkspace(t *testing.T) {
	repository := &fakeRepository{role: workspace.RoleViewer, member: false}
	service := NewService(repository)
	assigneeID := uuid.New()

	_, err := service.List(context.Background(), uuid.New(), uuid.New(), "website-redesign", ListFilter{AssigneeID: &assigneeID})

	require.ErrorIs(t, err, ErrInvalidAssignee)
}

func TestUpdateAllowsMemberOnlyForOwnAssignedIssue(t *testing.T) {
	actorID := uuid.New()
	repository := &fakeRepository{role: workspace.RoleMember, member: true, issue: Issue{ID: uuid.New(), AssigneeID: &actorID}}
	service := NewService(repository)

	updated, err := service.Update(context.Background(), actorID, uuid.New(), "website-redesign", repository.issue.ID, UpdateInput{
		Title:      "In progress",
		Status:     StatusInProgress,
		Priority:   PriorityMedium,
		AssigneeID: &actorID,
	})

	require.NoError(t, err)
	require.Equal(t, StatusInProgress, updated.Status)
}

func TestUpdateRejectsMemberForUnassignedIssue(t *testing.T) {
	repository := &fakeRepository{role: workspace.RoleMember, issue: Issue{ID: uuid.New()}}
	service := NewService(repository)

	_, err := service.Update(context.Background(), uuid.New(), uuid.New(), "website-redesign", repository.issue.ID, UpdateInput{Title: "Nope"})

	require.ErrorIs(t, err, ErrForbidden)
}

type fakeRepository struct {
	role      workspace.Role
	member    bool
	issue     Issue
	creatorID uuid.UUID
}

func (r *fakeRepository) Role(context.Context, uuid.UUID, uuid.UUID) (workspace.Role, error) {
	return r.role, nil
}
func (r *fakeRepository) Project(context.Context, uuid.UUID, string) (Project, error) {
	return Project{ID: uuid.New()}, nil
}
func (r *fakeRepository) Member(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return r.member, nil
}
func (r *fakeRepository) Create(_ context.Context, _ uuid.UUID, creatorID uuid.UUID, input CreateInput) (Issue, error) {
	r.creatorID = creatorID
	return Issue{ID: uuid.New(), Title: input.Title, Status: input.Status, Priority: input.Priority}, nil
}
func (r *fakeRepository) List(context.Context, uuid.UUID, ListFilter) ([]Issue, error) {
	return nil, nil
}
func (r *fakeRepository) Get(context.Context, uuid.UUID, uuid.UUID) (Issue, error) {
	return r.issue, nil
}
func (r *fakeRepository) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, input UpdateInput) (Issue, error) {
	r.issue.Title, r.issue.Status, r.issue.Priority = input.Title, input.Status, input.Priority
	return r.issue, nil
}
