package invite

import (
	"crypto/sha256"
	"testing"
)

func TestHashTokenIsDeterministicAndDoesNotReturnTheSecret(t *testing.T) {
	secret := "Q4D7CG1q5vZr8q4Mp45xqmhyhSK_Uyq3yn64Di2g5Cg"

	hash := hashToken(secret)

	want := sha256.Sum256([]byte(secret))
	if hash != want {
		t.Fatalf("hashToken(%q) = %x, want %x", secret, hash, want)
	}
}
