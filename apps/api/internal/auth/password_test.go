package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComparePasswordAcceptsOriginalPassword(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	require.NoError(t, ComparePassword(hash, password))
	require.Error(t, ComparePassword(hash, "a different password"))
}

func TestHashPasswordUsesUniqueSalts(t *testing.T) {
	first, err := HashPassword("correct horse battery staple")
	require.NoError(t, err)
	second, err := HashPassword("correct horse battery staple")
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestHashPasswordRejectsInvalidLength(t *testing.T) {
	_, err := HashPassword("short")
	require.ErrorIs(t, err, ErrInvalidPassword)

	_, err = HashPassword(strings.Repeat("a", 129))
	require.ErrorIs(t, err, ErrInvalidPassword)
}
