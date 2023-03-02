package storage

import (
	"context"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"go.infratographer.com/x/crdbx"

	"go.infratographer.com/identity-api/internal/types"
)

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
	types.OAuthClientManager
	TransactionManager
}

// EngineOption defines an initialization option for a storage engine.
type EngineOption func(*engine) error

// WithTracing enables tracing for the storage engine.
func WithTracing(config crdbx.Config) EngineOption {
	return func(e *engine) error {
		// Shut down the old database and replace it with a new one
		err := e.db.Close()
		if err != nil {
			return err
		}

		db, err := crdbx.NewDB(config, true)
		if err != nil {
			return err
		}

		e.db = db

		return nil
	}
}

// WithSeedData adds seed data to the storage engine.
func WithSeedData(data SeedData) EngineOption {
	return func(e *engine) error {
		return e.seedDatabase(context.Background(), data)
	}
}

// WithMigrations runs migrations on the storage engine.
func WithMigrations() EngineOption {
	return func(e *engine) error {
		return runMigrations(e.db)
	}
}

// NewEngine creates a new storage engine based on the given config.
func NewEngine(config crdbx.Config, opts ...EngineOption) (Engine, error) {
	return newCRDBEngine(config, opts...)
}

func buildIssuerFromSeed(seed SeedIssuer) (types.Issuer, error) {
	claimMappings, err := types.NewClaimsMapping(seed.ClaimMappings)
	if err != nil {
		return types.Issuer{}, err
	}

	out := types.Issuer{
		TenantID:      seed.TenantID,
		ID:            seed.ID,
		Name:          seed.Name,
		URI:           seed.URI,
		JWKSURI:       seed.JWKSURI,
		ClaimMappings: claimMappings,
	}

	return out, nil
}

// InMemoryCRDB creates an in-memory CRDB test server.
func InMemoryCRDB() (testserver.TestServer, error) {
	ts, err := testserver.NewTestServer()
	if err != nil {
		return nil, err
	}

	if err := ts.Start(); err != nil {
		return nil, err
	}

	return ts, nil
}
