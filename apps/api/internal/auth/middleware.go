package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Identity struct{ UserID uuid.UUID }

type identityContextKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}

func (s *Service) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
			return
		}
		identity, err := s.authenticate(r.Context(), token)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityContextKey{}, identity)))
	})
}

func (s *Service) authenticate(ctx context.Context, accessToken string) (Identity, error) {
	claims, err := s.tokens.ParseAccessToken(accessToken)
	if err != nil || !s.repository.HasActiveSession(ctx, claims.SessionID, claims.UserID) {
		return Identity{}, ErrUnauthenticated
	}
	return Identity{UserID: claims.UserID}, nil
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
