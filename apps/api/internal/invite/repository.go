package invite

import "crypto/sha256"

func hashToken(secret string) [sha256.Size]byte {
	return sha256.Sum256([]byte(secret))
}
