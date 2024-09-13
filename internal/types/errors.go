package types

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound represents an error condition where a resource was not found.
	ErrNotFound = errors.New("not found")
	// ErrInvalidArgument represents an error condition where an argument was invalid.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrorIssuerNotFound represents an error condition where an issuer was not found.
	ErrorIssuerNotFound = errors.New("issuer not found")

	// ErrUserInfoNotFound is returned if we attempt to fetch user info
	// from the storage backend and no info exists for that user.
	ErrUserInfoNotFound = errors.New("user info does not exist")

	// ErrFetchUserInfo represents a failure when making a /userinfo request.
	ErrFetchUserInfo = errors.New("could not fetch user info")

	// ErrInvalidUserInfo represents an error condition where the
	// UserInfo provided fails validation prior to storage.
	ErrInvalidUserInfo = errors.New("failed to store user info")

	// ErrOAuthClientNotFound is returned if the OAuthClient doesn't exist.
	ErrOAuthClientNotFound = errors.New("oauth client does not exist")

	// ErrGroupNotFound is returned if the group doesn't exist.
	ErrGroupNotFound = fmt.Errorf("%w: group not found", ErrNotFound)

	// ErrGroupExists is returned if the group already exists.
	ErrGroupExists = fmt.Errorf("%w: group already exists", ErrInvalidArgument)

	// ErrGroupNameEmpty is returned if the group name is empty.
	ErrGroupNameEmpty = fmt.Errorf("%w: group name is empty", ErrInvalidArgument)
)

// ErrorInvalidTokenRequest represents an error where an access token request failed.
type ErrorInvalidTokenRequest struct {
	Subject map[string]string
}

func (e ErrorInvalidTokenRequest) Error() string {
	return fmt.Sprintf("invalid access request for subject %v", e.Subject)
}
