package storage

import "context"

// SeedIssuer represents the seed data for a single issuer.
type SeedIssuer struct {
	TenantID      string
	ID            string
	Name          string
	URI           string
	JWKSURI       string
	ClaimMappings map[string]string
}

// SeedData represents the seed data for an identity-api instance on startup.
type SeedData struct {
	Issuers []SeedIssuer
}

// SeedDatabase seeds the database using the given storage config.
func SeedDatabase(ctx context.Context, config Config) error {
	switch config.Type {
	case "":
		return ErrorMissingEngineType
	case EngineTypeCRDB:
	default:
		err := &ErrorUnsupportedEngineType{
			engineType: config.Type,
		}

		return err
	}

	engine, err := newCRDBEngine(config)
	if err != nil {
		return err
	}

	return engine.seedDatabase(ctx, config.SeedData)
}
