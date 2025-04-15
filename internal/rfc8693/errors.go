package rfc8693

import (
	"errors"
	"fmt"
)

var (
	// ErrMissingSub represents an error where the 'sub' claim is missing from the input claims.
	ErrMissingSub = &ErrMissingClaim{
		claim: "sub",
	}

	// ErrMissingIss represents an error where the 'iss' claim is missing from the input claims.
	ErrMissingIss = &ErrMissingClaim{
		claim: "iss",
	}

	// ErrInvalidClaimCondition represents an error where the claim condition expression is invalid.
	ErrInvalidClaimCondition = errors.New("invalid claim condition expression")
)

// ErrMissingClaim represents an error where a required claim is missing.
type ErrMissingClaim struct {
	claim string
}

func (e *ErrMissingClaim) Error() string {
	return fmt.Sprintf("missing required claim '%s'", e.claim)
}
