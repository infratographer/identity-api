package storage

import (
	"context"
	"database/sql"

	"go.infratographer.com/x/crdbx"
)

type crdbEngine struct {
	*issuerService
	*userInfoService
	db *sql.DB
}

func newCRDBEngine(config Config) (*crdbEngine, error) {
	db, err := crdbx.NewDB(config.CRDB, config.Tracing)
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

	out := &crdbEngine{
		issuerService:   issSvc,
		userInfoService: userInfoSvc,
		db:              db,
	}

	return out, nil
}

func (eng *crdbEngine) Shutdown() {
}

func (eng *crdbEngine) BeginContext(ctx context.Context) (context.Context, error) {
	return beginTxContext(ctx, eng.db)
}

func (eng *crdbEngine) CommitContext(ctx context.Context) error {
	return commitContextTx(ctx)
}

func (eng *crdbEngine) RollbackContext(ctx context.Context) error {
	return rollbackContextTx(ctx)
}

func (eng *crdbEngine) seedDatabase(ctx context.Context, data SeedData) error {
	return eng.issuerService.seedDatabase(ctx, data.Issuers)
}
