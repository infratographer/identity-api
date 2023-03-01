// Package crypto provides tools to generate random tokens
package crypto

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
)

// SecureToken is a randomly generated token.
type SecureToken string

// GenerateSecureToken creates a cryptographically random SecureToken.
func GenerateSecureToken() (SecureToken, error) {
	randomData := make([]byte, md5.Size)

	_, err := rand.Read(randomData)
	if err != nil {
		return SecureToken(""), err
	}

	hasher := md5.New()
	output := hasher.Sum(randomData)

	return SecureToken(hex.EncodeToString(output)), nil
}
