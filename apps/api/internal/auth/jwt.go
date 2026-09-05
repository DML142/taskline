package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const accessTokenLifetime = 15 * time.Minute

var ErrInvalidAccessToken = errors.New("invalid access token")

type TokenManager struct {
	secret   []byte
	issuer   string
	audience string
	now      func() time.Time
}

type AccessClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type AccessTokenClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func NewTokenManager(secret []byte, issuer, audience string, now func() time.Time) *TokenManager {
	return &TokenManager{secret: secret, issuer: issuer, audience: audience, now: now}
}

func (m *TokenManager) NewAccessToken(userID, sessionID uuid.UUID) (string, error) {
	now := m.now()
	claims := AccessClaims{
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{m.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) ParseAccessToken(rawToken string) (AccessTokenClaims, error) {
	claims := AccessClaims{}
	parsed, err := jwt.ParseWithClaims(rawToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidAccessToken
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience), jwt.WithTimeFunc(m.now))
	if err != nil || !parsed.Valid || claims.ExpiresAt == nil || claims.IssuedAt == nil {
		return AccessTokenClaims{}, ErrInvalidAccessToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return AccessTokenClaims{}, ErrInvalidAccessToken
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return AccessTokenClaims{}, ErrInvalidAccessToken
	}
	return AccessTokenClaims{UserID: userID, SessionID: sessionID}, nil
}
