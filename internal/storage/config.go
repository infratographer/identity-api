package storage

// CRDBConfig represents CRDB-specific configuration for identity-api.
type CRDBConfig struct {
	URI string
}

// Config represents the storage configuration for identity-api.
type Config struct {
	Type     EngineType
	CRDB     CRDBConfig
	SeedData SeedData
}
