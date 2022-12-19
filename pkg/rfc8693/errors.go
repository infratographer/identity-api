package rfc8693

import (
	"fmt"
)

var (
	// ErrorMissingSub represents an error where the 'sub' claim is missing from the input claims.
	ErrorMissingSub = &ErrorMissingClaim{
		claim: "sub",
	}

	// ErrorUnknownIssuer represents an error where the issuer is not configured
	// in the mapping.
	ErrorUnknownIssuer = fmt.Errorf("unknown issuer")
)

// ErrorCELParse represents an error during CEL parsing.
type ErrorCELParse struct {
	inner error
}

func (ErrorCELParse) Error() string {
	return "error parsing CEL expression"
}

// Is returns true if target is a *ErrorCELParse.
func (e *ErrorCELParse) Is(target error) bool {
	_, ok := target.(*ErrorCELParse)

	return ok
}

// Unwrap returns the inner error from CEL parsing.
func (e *ErrorCELParse) Unwrap() error {
	return e.inner
}

// ErrorCELEval represents an error during CEL evaluation.
type ErrorCELEval struct {
	inner error
}

func (ErrorCELEval) Error() string {
	return "error evaluating CEL expression"
}

// Is returns true if target is a *ErrorCELEval.
func (e *ErrorCELEval) Is(target error) bool {
	_, ok := target.(*ErrorCELEval)

	return ok
}

// Unwrap returns the inner error from CEL evaluation.
func (e *ErrorCELEval) Unwrap() error {
	return e.inner
}

// ErrorMissingClaim represents an error where a required claim is missing.
type ErrorMissingClaim struct {
	claim string
}

func (e *ErrorMissingClaim) Error() string {
	return fmt.Sprintf("missing required claim '%s'", e.claim)
}
