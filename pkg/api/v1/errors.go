package v1

import "fmt"

// ErrorIssuerNotFound represents an error condition where a given issuer was not found.
type ErrorIssuerNotFound struct {
	URI string
}

func (e *ErrorIssuerNotFound) Error() string {
	return fmt.Sprintf("issuer '%s' not found", e.URI)
}
