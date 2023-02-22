package storage

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/cockroach-go/v2/testserver"

	"go.infratographer.com/identity-api/internal/types"
)

type memoryEngine struct {
	*issuerService
	*userInfoService
	crdb testserver.TestServer
	db   *sql.DB
}

func newMemoryEngine(config Config) (*memoryEngine, error) {
	crdb, err := inMemoryCRDB()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", crdb.PGURL().String())
	if err != nil {
		return nil, err
	}

	err = RunMigrations(db)
	if err != nil {
		return nil, err
	}

	issSvc, err := newIssuerService(config, db)
	if err != nil {
		return nil, err
	}

	userInfoSvc, err := newUserInfoService(config, db)
	if err != nil {
		return nil, err
	}

	out := &memoryEngine{
		issuerService:   issSvc,
		userInfoService: userInfoSvc,
		crdb:            crdb,
		db:              db,
	}

	return out, nil
}

func (eng *memoryEngine) Shutdown() {
	eng.crdb.Stop()
}

func (eng *memoryEngine) BeginContext(ctx context.Context) (context.Context, error) {
	return beginTxContext(ctx, eng.db)
}

func (eng *memoryEngine) CommitContext(ctx context.Context) error {
	return commitContextTx(ctx)
}

func (eng *memoryEngine) RollbackContext(ctx context.Context) error {
	return rollbackContextTx(ctx)
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

func inMemoryCRDB() (testserver.TestServer, error) {
	ts, err := testserver.NewTestServer()
	if err != nil {
		return nil, err
	}

	if err := ts.Start(); err != nil {
		return nil, err
	}

	return ts, nil
}
