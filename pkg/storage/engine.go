package storage

import (
	"database/sql"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

const (
	// EngineTypeMemory represents an in-memory storage engine.
	EngineTypeMemory EngineType = "memory"
)

// EngineType represents the type of identity-manager-sts storage engine.
type EngineType string

// Engine represents a storage engine.
type Engine interface {
	v1.IssuerService
}

// NewEngine creates a new storage engine based on the given config.
func NewEngine(config Config) (Engine, error) {
	switch config.Type {
	case "":
		return nil, ErrorMissingEngineType
	case EngineTypeMemory:
		db, err := inMemoryCRDB()
		if err != nil {
			return nil, err
		}

		config.db = db

		issSvc, err := newMemoryIssuerService(config)
		if err != nil {
			return nil, err
		}

		out := &memoryEngine{
			memoryIssuerService: issSvc,
		}

		return out, nil
	default:
		err := &ErrorUnknownEngineType{
			engineType: config.Type,
		}

		return nil, err
	}
}

func inMemoryCRDB() (*sql.DB, error) {
	ts, err := testserver.NewTestServer()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", ts.PGURL().String())
	if err != nil {
		return nil, err
	}

	return db, nil
}
