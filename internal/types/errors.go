package types

import "errors"

var (
	// ErrorIssuerNotFound represents an error condition where an issuer was not found.
	ErrorIssuerNotFound = errors.New("issuer not found")
)
