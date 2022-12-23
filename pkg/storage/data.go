package storage

// SeedIssuer represents the seed data for a single issuer.
type SeedIssuer struct {
	ID            string
	Name          string
	URI           string
	JWKSURI       string
	ClaimMappings map[string]string
}

// SeedData represents the seed data for a DMV instance on startup.
type SeedData struct {
	Issuers []SeedIssuer
}
