package storage

import v1 "go.infratographer.com/dmv/pkg/api/v1"

// MemoryConfig represents the configuration for in-memory DMV storage.
type MemoryConfig struct {
	Issuers []v1.Issuer
}

// Config represents the storage configuration for DMV.
type Config struct {
	Type   EngineType
	Memory MemoryConfig
}
