package v1

// ErrorResponse represents a generic error response.
type ErrorResponse struct {
	Errors []string `json:"errors"`
}
