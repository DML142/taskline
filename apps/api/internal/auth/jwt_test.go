package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTokenManagerParsesValidAccessToken(t *testing.T) {
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	manager := NewTokenManager([]byte(strings.Repeat("k", 32)), "taskline-api", "taskline-web", func() time.Time { return now })
	userID := uuid.New()
	sessionID := uuid.New()
	token, err := manager.NewAccessToken(userID, sessionID)
	require.NoError(t, err)

	claims, err := manager.ParseAccessToken(token)

	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, sessionID, claims.SessionID)
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	issuer := NewTokenManager([]byte(strings.Repeat("k", 32)), "taskline-api", "taskline-web", func() time.Time { return now })
	token, err := issuer.NewAccessToken(uuid.New(), uuid.New())
	require.NoError(t, err)
	expired := NewTokenManager([]byte(strings.Repeat("k", 32)), "taskline-api", "taskline-web", func() time.Time { return now.Add(16 * time.Minute) })

	_, err = expired.ParseAccessToken(token)

	require.Error(t, err)
}

func TestTokenManagerRejectsWrongAudience(t *testing.T) {
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	issuer := NewTokenManager([]byte(strings.Repeat("k", 32)), "taskline-api", "taskline-web", func() time.Time { return now })
	token, err := issuer.NewAccessToken(uuid.New(), uuid.New())
	require.NoError(t, err)
	wrongAudience := NewTokenManager([]byte(strings.Repeat("k", 32)), "taskline-api", "other-web", func() time.Time { return now })

	_, err = wrongAudience.ParseAccessToken(token)

	require.Error(t, err)
}

func TestTokenManagerRejectsOtherSigningMethods(t *testing.T) {
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	manager := NewTokenManager([]byte(strings.Repeat("k", 32)), "taskline-api", "taskline-web", func() time.Time { return now })
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, AccessClaims{
		SessionID: uuid.NewString(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "taskline-api",
			Subject:   uuid.NewString(),
			Audience:  jwt.ClaimStrings{"taskline-web"},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}).SignedString([]byte(strings.Repeat("k", 32)))
	require.NoError(t, err)

	_, err = manager.ParseAccessToken(token)

	require.ErrorIs(t, err, ErrInvalidAccessToken)
}
