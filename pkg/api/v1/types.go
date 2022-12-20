package v1

import "context"

// Issuer represents a token issuer.
type Issuer struct {
	// ID represents the ID of the issuer in DMV.
	ID string
	// Name represents the human-readable name of the issuer.
	Name string
	// URI represents the issuer URI as found in the "iss" claim of a JWT.
	URI string
	// JWKSURI represents the URI where the issuer's JWKS lives. Must be accessible by DMV.
	JWKSURI string
}

// IssuerService represents a service for managing issuers.
type IssuerService interface {
	GetByURI(ctx context.Context, uri string) (*Issuer, error)
}
