package store

import (
	"crypto/sha256"

	"github.com/google/uuid"
)

// GenerateInvitationToken generates a random UUIDv4 string and returns:
// 1. The plaintext token string (to be passed to the email template).
// 2. The SHA-256 hash byte slice (to be stored in the user_invitations BYTEA column).
func GenerateInvitationToken() (string, []byte) {
	plainToken := uuid.New().String()
	tokenHash := sha256.Sum256([]byte(plainToken))

	return plainToken, tokenHash[:]
}