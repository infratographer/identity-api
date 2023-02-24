package storage

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/ory/fosite"

	"go.infratographer.com/identity-api/internal/types"
)

type memoryEngine struct {
	*issuerService
	*userInfoService
	crdb testserver.TestServer
	db   *sql.DB
}

// CreateAccessTokenSession implements oauth2.AccessTokenStorage
func (memoryEngine) CreateAccessTokenSession(ctx context.Context, signature string, request fosite.Requester) (err error) {
	panic("unimplemented")
}

// DeleteAccessTokenSession implements oauth2.AccessTokenStorage
func (memoryEngine) DeleteAccessTokenSession(ctx context.Context, signature string) (err error) {
	panic("unimplemented")
}

// GetAccessTokenSession implements oauth2.AccessTokenStorage
func (memoryEngine) GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (request fosite.Requester, err error) {
	panic("unimplemented")
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

	err = runMigrations(db)
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

	err = out.seedDatabase(context.Background(), config.SeedData)
	if err != nil {
		return nil, err
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

func (eng *memoryEngine) seedDatabase(ctx context.Context, data SeedData) error {
	return eng.issuerService.seedDatabase(ctx, data.Issuers)
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
