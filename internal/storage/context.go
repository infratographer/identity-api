package storage

import (
	"context"
	"database/sql"
)

type contextKey int

var txKey contextKey

func beginTxContext(ctx context.Context, db *sql.DB) (context.Context, error) {
	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return nil, err
	}

	out := context.WithValue(ctx, txKey, tx)

	return out, nil
}

func getContextTx(ctx context.Context) (*sql.Tx, error) {
	maybeTx := ctx.Value(txKey)

	tx, ok := maybeTx.(*sql.Tx)
	if !ok {
		return nil, ErrorMissingContextTx
	}

	return tx, nil
}

func commitContextTx(ctx context.Context) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func rollbackContextTx(ctx context.Context) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	return tx.Rollback()
}
