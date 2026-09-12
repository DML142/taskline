package comment

import (
	"testing"

	"github.com/stretchr/testify/require"

	"taskline/apps/api/internal/auth"
)

func TestNewHandlerRegistersCommentRoutes(t *testing.T) {
	authentication := auth.NewService(nil, auth.NewTokenManager([]byte("01234567890123456789012345678901"), "taskline-api", "taskline-web", nil), nil)

	require.NotPanics(t, func() { NewHandler(nil, authentication) })
}
