package storage

import "go.infratographer.com/x/crdbx"

// Config represents the storage configuration for identity-api.
type Config struct {
	Type     EngineType
	Tracing  bool
	CRDB     crdbx.Config
	SeedData SeedData
}
