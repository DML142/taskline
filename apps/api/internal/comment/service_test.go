package comment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/workspace"
)

func TestCreateRejectsViewer(t *testing.T) {
	service := NewService(&fakeRepository{role: workspace.RoleViewer})

	_, err := service.Create(context.Background(), uuid.New(), uuid.New(), "website", uuid.New(), CreateInput{Body: "I found a regression."})

	require.ErrorIs(t, err, ErrForbidden)
}

func TestUpdateRejectsAnotherAuthorsComment(t *testing.T) {
	authorID := uuid.New()
	commentID := uuid.New()
	service := NewService(&fakeRepository{
		role:    workspace.RoleMember,
		comment: Comment{ID: commentID, AuthorID: authorID},
	})

	_, err := service.Update(context.Background(), uuid.New(), uuid.New(), "website", uuid.New(), commentID, UpdateInput{Body: "Changed wording."})

	require.ErrorIs(t, err, ErrForbidden)
}

func TestCreateNormalizesCommentBody(t *testing.T) {
	repository := &fakeRepository{role: workspace.RoleMember}
	service := NewService(repository)

	_, err := service.Create(context.Background(), uuid.New(), uuid.New(), "website", uuid.New(), CreateInput{Body: "  Ready to release.  "})

	require.NoError(t, err)
	require.Equal(t, "Ready to release.", repository.createdInput.Body)
}

func TestCreateRejectsBlankComment(t *testing.T) {
	service := NewService(&fakeRepository{role: workspace.RoleAdmin})

	_, err := service.Create(context.Background(), uuid.New(), uuid.New(), "website", uuid.New(), CreateInput{Body: " \n "})

	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestDeleteAllowsOnlyTheAuthor(t *testing.T) {
	actorID := uuid.New()
	commentID := uuid.New()
	repository := &fakeRepository{
		role:    workspace.RoleAdmin,
		comment: Comment{ID: commentID, AuthorID: actorID},
	}
	service := NewService(repository)

	err := service.Delete(context.Background(), actorID, uuid.New(), "website", uuid.New(), commentID)

	require.NoError(t, err)
	require.Equal(t, commentID, repository.deletedID)
}

func TestUpdateRejectsBlankBodyBeforeWriting(t *testing.T) {
	actorID := uuid.New()
	commentID := uuid.New()
	repository := &fakeRepository{
		role:    workspace.RoleMember,
		comment: Comment{ID: commentID, AuthorID: actorID},
	}
	service := NewService(repository)

	_, err := service.Update(context.Background(), actorID, uuid.New(), "website", uuid.New(), commentID, UpdateInput{Body: " "})

	require.ErrorIs(t, err, ErrInvalidRequest)
	require.False(t, repository.updated)
}

func TestListAllowsViewerToReadIssueComments(t *testing.T) {
	comments := []Comment{{ID: uuid.New(), Body: "Please check mobile."}}
	service := NewService(&fakeRepository{role: workspace.RoleViewer, comments: comments})

	actual, err := service.List(context.Background(), uuid.New(), uuid.New(), "website", uuid.New())

	require.NoError(t, err)
	require.Equal(t, comments, actual)
}

type fakeRepository struct {
	role         workspace.Role
	comment      Comment
	createdInput CreateInput
	deletedID    uuid.UUID
	updated      bool
	comments     []Comment
}

func (r *fakeRepository) Role(context.Context, uuid.UUID, uuid.UUID) (workspace.Role, error) {
	return r.role, nil
}

func (*fakeRepository) Project(context.Context, uuid.UUID, string) (Project, error) {
	return Project{ID: uuid.New()}, nil
}

func (*fakeRepository) Issue(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (r *fakeRepository) Create(_ context.Context, _ uuid.UUID, _ uuid.UUID, input CreateInput) (Comment, error) {
	r.createdInput = input
	return Comment{}, nil
}

func (r *fakeRepository) Get(context.Context, uuid.UUID, uuid.UUID) (Comment, error) {
	return r.comment, nil
}

func (r *fakeRepository) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ UpdateInput) (Comment, error) {
	r.updated = true
	return Comment{}, nil
}

func (r *fakeRepository) Delete(_ context.Context, _ uuid.UUID, commentID uuid.UUID) error {
	r.deletedID = commentID
	return nil
}

func (r *fakeRepository) List(context.Context, uuid.UUID) ([]Comment, error) { return r.comments, nil }
