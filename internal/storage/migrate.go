package storage

import (
	"database/sql"
	"embed"
	"sync"

	"github.com/pressly/goose/v3"
	"go.infratographer.com/x/crdbx"
)

//go:embed migrations/*
var embedMigrations embed.FS

var migrationMutex sync.Mutex

func init() {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
}

// RunMigrations runs all migrations using the given storage config.
func RunMigrations(config crdbx.Config) error {
	db, err := crdbx.NewDB(config, false)
	if err != nil {
		return err
	}

	defer db.Close()

	return runMigrations(db)
}

// runMigrations runs all embedded migrations against the given database. Subsequent calls
// will have no effect. This function is safe to run across multiple goroutines.
func runMigrations(db *sql.DB) error {
	migrationMutex.Lock()
	defer migrationMutex.Unlock()

	err := goose.Up(db, "migrations")

	return err
}
