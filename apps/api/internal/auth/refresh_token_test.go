package auth

import "testing"

import "github.com/stretchr/testify/require"

func TestNewRefreshTokenIsRandomAndHashIsStable(t *testing.T) {
	first, err := NewRefreshToken()
	require.NoError(t, err)
	second, err := NewRefreshToken()
	require.NoError(t, err)

	require.NotEqual(t, first, second)
	require.Equal(t, HashRefreshToken(first), HashRefreshToken(first))
	require.NotEqual(t, HashRefreshToken(first), HashRefreshToken(second))
}
