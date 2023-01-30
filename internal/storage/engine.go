package storage

import (
	"context"
	"database/sql"

	"go.infratographer.com/identity-manager-sts/internal/types"
)

const (
	// EngineTypeMemory represents an in-memory storage engine.
	EngineTypeMemory EngineType = "memory"
)

// EngineType represents the type of identity-manager-sts storage engine.
type EngineType string

// Engine represents a storage engine.
type Engine interface {
	types.IssuerService
	BeginContext(context.Context) (context.Context, error)
	CommitContext(context.Context) error
	RollbackContext(context.Context) error
	Shutdown()
}

// NewEngine creates a new storage engine based on the given config.
func NewEngine(config Config) (Engine, error) {
	switch config.Type {
	case "":
		return nil, ErrorMissingEngineType
	case EngineTypeMemory:
		crdb, err := inMemoryCRDB()
		if err != nil {
			return nil, err
		}

		db, err := sql.Open("postgres", crdb.PGURL().String())
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
			crdb:                crdb,
			db:                  db,
		}

		return out, nil
	default:
		err := &ErrorUnknownEngineType{
			engineType: config.Type,
		}

		return nil, err
	}
}
