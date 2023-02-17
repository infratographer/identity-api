package storage

import (
	"database/sql"
)

// Config represents the storage configuration for identity-api.
type Config struct {
	Type     EngineType
	SeedData SeedData

	db *sql.DB
}
