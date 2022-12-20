package storage

import (
	"context"

	v1 "go.infratographer.com/dmv/pkg/api/v1"
)

type memoryEngine struct {
	*memoryIssuerService
}

// memoryIssuerService represents an in-memory issuer service.
type memoryIssuerService struct {
	issuers map[string]v1.Issuer
}

// newMemoryIssuerService creates a new in-memory issuer service.
func newMemoryIssuerService(config MemoryConfig) (*memoryIssuerService, error) {
	issuerMap := make(map[string]v1.Issuer, len(config.Issuers))
	for _, iss := range config.Issuers {
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
