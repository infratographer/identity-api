package storage

import (
	"context"
	"database/sql"

	"go.infratographer.com/x/crdbx"
)

type engine struct {
	*issuerService
	*userInfoService
	*oauthClientManager
	db *sql.DB
}

func newCRDBEngine(config crdbx.Config, options ...EngineOption) (*engine, error) {
	db, err := crdbx.NewDB(config, false)
	if err != nil {
		return nil, err
	}

	issSvc, err := newIssuerService(db)
	if err != nil {
		return nil, err
	}

	userInfoSvc, err := newUserInfoService(db)
	if err != nil {
		return nil, err
	}

	oauthClientManager, err := newOAuthClientManager(db)
	if err != nil {
		return nil, err
	}

	out := &engine{
		issuerService:      issSvc,
		userInfoService:    userInfoSvc,
		oauthClientManager: oauthClientManager,
		db:                 db,
	}

	for _, opt := range options {
		err = opt(out)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (eng *engine) Shutdown() {
}

func (eng *engine) BeginContext(ctx context.Context) (context.Context, error) {
	return beginTxContext(ctx, eng.db)
}

func (eng *engine) CommitContext(ctx context.Context) error {
	return commitContextTx(ctx)
}

func (eng *engine) RollbackContext(ctx context.Context) error {
	return rollbackContextTx(ctx)
}

func (eng *engine) seedDatabase(ctx context.Context, data SeedData) error {
	return eng.issuerService.seedDatabase(ctx, data.Issuers)
}
