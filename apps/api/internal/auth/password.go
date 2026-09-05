package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	passwordMemory      = 19 * 1024
	passwordIterations  = 2
	passwordParallelism = 1
	passwordSaltLength  = 16
	passwordKeyLength   = 32
)

var (
	ErrInvalidPassword  = errors.New("invalid password")
	errPasswordMismatch = errors.New("password mismatch")
)

func HashPassword(password string) (string, error) {
	if !validPassword(password) {
		return "", ErrInvalidPassword
	}
	salt := make([]byte, passwordSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemory, passwordParallelism, passwordKeyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", passwordMemory, passwordIterations, passwordParallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func ComparePassword(encodedHash, password string) error {
	if !validPassword(password) {
		return errPasswordMismatch
	}
	salt, expected, err := parsePasswordHash(encodedHash)
	if err != nil {
		return errPasswordMismatch
	}
	actual := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemory, passwordParallelism, passwordKeyLength)
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return errPasswordMismatch
	}
	return nil
}

func validPassword(password string) bool {
	return utf8.ValidString(password) && utf8.RuneCountInString(password) >= 12 && utf8.RuneCountInString(password) <= 128
}

func parsePasswordHash(encodedHash string) ([]byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", passwordMemory, passwordIterations, passwordParallelism) {
		return nil, nil, errors.New("invalid password hash")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != passwordSaltLength {
		return nil, nil, errors.New("invalid password hash")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) != passwordKeyLength {
		return nil, nil, errors.New("invalid password hash")
	}
	return salt, hash, nil
}
