// Package jwks provides a fosite.IssuerJWKSURIStrategy implementation.
package jwks

import (
	"context"
	"fmt"

	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
	"go.infratographer.com/identity-manager-sts/internal/fositex"
)

var (
	// ErrUnknownIssuer is returned when the issuer is unknown.
	ErrUnknownIssuer = fmt.Errorf("unknown JWT issuer")
)

type issuerJWKSURIStrategy struct {
	issuerSvc v1.IssuerService
}

// NewIssuerJWKSURIStrategy creates a new fosite.IssuerJWKSURIStrategy.
func NewIssuerJWKSURIStrategy(issuerSvc v1.IssuerService) fositex.IssuerJWKSURIStrategy {
	out := issuerJWKSURIStrategy{
		issuerSvc: issuerSvc,
	}

	return out
}

func (s issuerJWKSURIStrategy) GetIssuerJWKSURI(ctx context.Context, iss string) (string, error) {
	issuer, err := s.issuerSvc.GetByURI(ctx, iss)
	if err != nil {
		return "", err
	}

	return issuer.JWKSURI, nil
}
