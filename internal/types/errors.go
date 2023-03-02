package types

import "errors"

var (
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
)
