package storage

import (
	"context"

	v1 "go.infratographer.com/dmv/pkg/api/v1"
)

type memoryEngine struct {
	*memoryIssuerService
}

func buildIssuerFromSeed(seed SeedIssuer) (v1.Issuer, error) {
	claimMappings, err := v1.BuildClaimsMappingFromMap(seed.ClaimMappings)
	if err != nil {
		return v1.Issuer{}, err
	}

	out := v1.Issuer{
		ID:            seed.ID,
		Name:          seed.Name,
		URI:           seed.URI,
		JWKSURI:       seed.JWKSURI,
		ClaimMappings: claimMappings,
	}

	return out, nil
}

// memoryIssuerService represents an in-memory issuer service.
type memoryIssuerService struct {
	issuers map[string]v1.Issuer
}

// newMemoryEngine creates a new in-memory storage engine.
func newMemoryIssuerService(config Config) (*memoryIssuerService, error) {
	issuerMap := make(map[string]v1.Issuer, len(config.SeedData.Issuers))

	for _, seed := range config.SeedData.Issuers {
		iss, err := buildIssuerFromSeed(seed)
		if err != nil {
			return nil, err
		}

		issuerMap[iss.URI] = iss
	}

	out := &memoryIssuerService{
		issuers: issuerMap,
	}

	return out, nil
}

// GetByURI looks up the given issuer by URI, returning the issuer if one exists.
func (s *memoryIssuerService) GetByURI(ctx context.Context, uri string) (*v1.Issuer, error) {
	iss, ok := s.issuers[uri]
	if !ok {
		err := v1.ErrorIssuerNotFound{
			URI: uri,
		}

		return nil, err
	}

	return &iss, nil
}
