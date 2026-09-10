package issue

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestListFilterParsesStatusAssigneeAndPriority(t *testing.T) {
	assigneeID := uuid.New()
	request := httptest.NewRequest("GET", "/?status=IN_PROGRESS&assigneeId="+assigneeID.String()+"&priority=HIGH", nil)

	filter, err := listFilter(request)

	require.NoError(t, err)
	require.Equal(t, StatusInProgress, filter.Status)
	require.Equal(t, &assigneeID, filter.AssigneeID)
	require.Equal(t, PriorityHigh, filter.Priority)
}

func TestListFilterRejectsInvalidAssignee(t *testing.T) {
	request := httptest.NewRequest("GET", "/?assigneeId=not-a-uuid", nil)

	_, err := listFilter(request)

	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestListFilterRejectsInvalidPriority(t *testing.T) {
	_, err := listFilter(httptest.NewRequest("GET", "/?priority=URGENT", nil))

	require.ErrorIs(t, err, ErrInvalidRequest)
}
