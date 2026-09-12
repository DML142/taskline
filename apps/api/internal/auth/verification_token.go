package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func NewEmailVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate email verification token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashEmailVerificationToken(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}
