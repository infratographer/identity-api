// Package crypto provides tools to generate random tokens
package crypto

import (
	"github.com/ory/x/randx"
)

var secretCharSet = randx.AlphaNum

// SecureToken is a randomly generated token.
type SecureToken string

// GenerateSecureToken creates a cryptographically random SecureToken.
func GenerateSecureToken(length int) (SecureToken, error) {
	secret, err := randx.RuneSequence(length, secretCharSet)
	if err != nil {
		return SecureToken(""), err
	}

	return SecureToken(secret), nil
}
