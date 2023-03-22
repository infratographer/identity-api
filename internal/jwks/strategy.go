// Package jwks provides a fositex.IssuerJWKSURIProvider implementation.
package jwks

import (
	"context"
	"fmt"

	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/types"
)

var (
	// ErrUnknownIssuer is returned when the issuer is unknown.
	ErrUnknownIssuer = fmt.Errorf("unknown JWT issuer")
)

type issuerJWKSURIProvider struct {
	issuerSvc types.IssuerService
}

// NewIssuerJWKSURIProvider creates a new fositex.IssuerJWKSURIProvider.
func NewIssuerJWKSURIProvider(issuerSvc types.IssuerService) fositex.IssuerJWKSURIProvider {
	out := issuerJWKSURIProvider{
		issuerSvc: issuerSvc,
	}

	return out
}

func (s issuerJWKSURIProvider) GetIssuerJWKSURI(ctx context.Context, iss string) (string, error) {
	issuer, err := s.issuerSvc.GetIssuerByURI(ctx, iss)
	if err != nil {
		return "", err
	}

	return issuer.JWKSURI, nil
}
