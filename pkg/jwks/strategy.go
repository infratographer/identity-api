package jwks

import (
	"context"
	"fmt"
	"go.infratographer.com/dmv/pkg/fositex"
)

type issuerJWKSURIStrategy struct {
	issuerURIs map[string]string
}

func NewIssuerJWKSURIStrategy(issuers []fositex.Issuer) fositex.IssuerJWKSURIStrategy {
	issuerURIs := make(map[string]string)
	for _, iss := range issuers {
		issuerURIs[iss.Name] = iss.JWKSURI
	}

	out := issuerJWKSURIStrategy{
		issuerURIs: issuerURIs,
	}

	return out
}

func (s issuerJWKSURIStrategy) GetIssuerJWKSURI(ctx context.Context, iss string) (string, error) {
	jwksURI, ok := s.issuerURIs[iss]
	if !ok {
		return "", fmt.Errorf("Unknown JWT issuer '%s'.", iss)
	}

	return jwksURI, nil
}
