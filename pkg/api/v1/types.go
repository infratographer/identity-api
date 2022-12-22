package v1

import (
	"context"
	"encoding/json"

	"github.com/google/cel-go/cel"

	"go.infratographer.com/dmv/pkg/celutils"
)

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
	// ClaimMappings represents a map of claims to a CEL expression that will be evaluated
	ClaimMappings ClaimsMapping
}

// IssuerService represents a service for managing issuers.
type IssuerService interface {
	GetByURI(ctx context.Context, uri string) (*Issuer, error)
}

// ClaimsMapping represents a map of claims to a CEL expression that will be evaluated
type ClaimsMapping map[string]StringToCEL

// StringToCEL represents a string that is a CEL expression and the compiled program.
type StringToCEL struct {
	Repr    string
	Program cel.Program
}

// MarshalJSON implements the json.Marshaler interface.
func (c ClaimsMapping) MarshalJSON() ([]byte, error) {
	out := make(map[string]string, len(c))
	for k, v := range c {
		out[k] = v.Repr
	}

	return json.Marshal(out)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *ClaimsMapping) UnmarshalJSON(data []byte) error {
	in := make(map[string]string)
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}

	out, err := BuildClaimsMappingFromMap(in)
	if err != nil {
		return err
	}

	*c = out

	return nil
}

// BuildClaimsMappingFromMap builds a ClaimsMapping from a map of strings.
func BuildClaimsMappingFromMap(in map[string]string) (ClaimsMapping, error) {
	out := make(ClaimsMapping, len(in))

	for k, v := range in {
		prog, err := celutils.ParseCEL(v)
		if err != nil {
			return nil, err
		}

		out[k] = StringToCEL{
			Repr:    v,
			Program: prog,
		}
	}

	return out, nil
}
