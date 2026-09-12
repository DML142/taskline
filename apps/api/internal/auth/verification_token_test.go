package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEmailVerificationTokenIsRandomAndHashesDeterministically(t *testing.T) {
	first, err := NewEmailVerificationToken()
	require.NoError(t, err)
	second, err := NewEmailVerificationToken()
	require.NoError(t, err)

	require.NotEmpty(t, first)
	require.NotEqual(t, first, second)
	require.Equal(t, HashEmailVerificationToken(first), HashEmailVerificationToken(first))
	require.NotEqual(t, HashEmailVerificationToken(first), HashEmailVerificationToken(second))
}
