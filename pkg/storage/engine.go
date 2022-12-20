package storage

import (
	v1 "go.infratographer.com/dmv/pkg/api/v1"
)

const (
	// EngineTypeMemory represents an in-memory storage engine.
	EngineTypeMemory EngineType = "memory"
)

// EngineType represents the type of DMV storage engine.
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
		issSvc, err := newMemoryIssuerService(config.Memory)
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
