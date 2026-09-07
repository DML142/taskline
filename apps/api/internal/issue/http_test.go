package issue

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestListFilterParsesStatusAndAssignee(t *testing.T) {
	assigneeID := uuid.New()
	request := httptest.NewRequest("GET", "/?status=IN_PROGRESS&assigneeId="+assigneeID.String(), nil)

	filter, err := listFilter(request)

	require.NoError(t, err)
	require.Equal(t, StatusInProgress, filter.Status)
	require.Equal(t, &assigneeID, filter.AssigneeID)
}

func TestListFilterRejectsInvalidAssignee(t *testing.T) {
	request := httptest.NewRequest("GET", "/?assigneeId=not-a-uuid", nil)

	_, err := listFilter(request)

	require.ErrorIs(t, err, ErrInvalidRequest)
}
