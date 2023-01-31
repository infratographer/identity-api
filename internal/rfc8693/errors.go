package rfc8693

import (
	"fmt"
)

var (
	// ErrorMissingSub represents an error where the 'sub' claim is missing from the input claims.
	ErrorMissingSub = &ErrorMissingClaim{
		claim: "sub",
	}

	// ErrorMissingIss represents an error where the 'iss' claim is missing from the input claims.
	ErrorMissingIss = &ErrorMissingClaim{
		claim: "iss",
	}
)

// ErrorMissingClaim represents an error where a required claim is missing.
type ErrorMissingClaim struct {
	claim string
}

func (e *ErrorMissingClaim) Error() string {
	return fmt.Sprintf("missing required claim '%s'", e.claim)
}
