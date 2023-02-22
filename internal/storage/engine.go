package storage

import (
	"context"

	"go.infratographer.com/identity-api/internal/types"
)

const (
	// EngineTypeMemory represents an in-memory storage engine.
	EngineTypeMemory EngineType = "memory"

	// EngineTypeCRDB represents an external CockroachDB storage engine.
	EngineTypeCRDB EngineType = "crdb"
)

// EngineType represents the type of identity-api storage engine.
type EngineType string

// TransactionManager manages the state of sql transactions within a context
type TransactionManager interface {
	BeginContext(context.Context) (context.Context, error)
	CommitContext(context.Context) error
	RollbackContext(context.Context) error
}

// Engine represents a storage engine.
type Engine interface {
	types.IssuerService
	types.UserInfoService
	TransactionManager
	Shutdown()
}

// NewEngine creates a new storage engine based on the given config.
func NewEngine(config Config) (Engine, error) {
	switch config.Type {
	case "":
		return nil, ErrorMissingEngineType
	case EngineTypeMemory:
		return newMemoryEngine(config)
	case EngineTypeCRDB:
		return newCRDBEngine(config)
	default:
		err := &ErrorUnknownEngineType{
			engineType: config.Type,
		}

		return nil, err
	}
}
