// Package jwks provides a fosite.IssuerJWKSURIStrategy implementation.
package jwks

import (
	"context"
	"fmt"

	"go.infratographer.com/identity-manager-sts/internal/fositex"
	"go.infratographer.com/identity-manager-sts/internal/types"
)

var (
	// ErrUnknownIssuer is returned when the issuer is unknown.
	ErrUnknownIssuer = fmt.Errorf("unknown JWT issuer")
)

type issuerJWKSURIStrategy struct {
	issuerSvc types.IssuerService
}

// NewIssuerJWKSURIStrategy creates a new fosite.IssuerJWKSURIStrategy.
func NewIssuerJWKSURIStrategy(issuerSvc types.IssuerService) fositex.IssuerJWKSURIStrategy {
	out := issuerJWKSURIStrategy{
		issuerSvc: issuerSvc,
	}

	return out
}

func (s issuerJWKSURIStrategy) GetIssuerJWKSURI(ctx context.Context, iss string) (string, error) {
	issuer, err := s.issuerSvc.GetIssuerByURI(ctx, iss)
	if err != nil {
		return "", err
	}

	return issuer.JWKSURI, nil
}
