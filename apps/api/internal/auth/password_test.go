package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComparePasswordAcceptsOriginalPassword(t *testing.T) {
	password := "correct horse battery staple1"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	require.NoError(t, ComparePassword(hash, password))
	require.Error(t, ComparePassword(hash, "a different password"))
}

func TestHashPasswordUsesUniqueSalts(t *testing.T) {
	first, err := HashPassword("correct horse battery staple1")
	require.NoError(t, err)
	second, err := HashPassword("correct horse battery staple1")
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestHashPasswordEnforcesPasswordPolicy(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{name: "minimum Unicode length", password: "пароль12", valid: true},
		{name: "maximum Unicode length", password: "A1" + strings.Repeat("bc", 15), valid: true},
		{name: "too short", password: "Abcdef1", valid: false},
		{name: "too long", password: "A1" + strings.Repeat("bc", 15) + "d", valid: false},
		{name: "no digit", password: "Abcdefgh", valid: false},
		{name: "no letter", password: "12345678", valid: false},
		{name: "four repeated characters", password: "Abc1111d", valid: false},
		{name: "more than bcrypt byte limit", password: strings.Repeat("я", 36) + "A1", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HashPassword(tt.password)
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrInvalidPassword)
		})
	}
}
