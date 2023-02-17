package storage

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
