package storage

import (
	"database/sql"
	"embed"
	"sync"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var embedMigrations embed.FS

var migrationWG sync.WaitGroup

func init() {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
}

// RunMigrations runs all migrations using the given storage config.
func RunMigrations(config Config) error {
	switch config.Type {
	case "":
		return ErrorMissingEngineType
	case EngineTypeCRDB:
	default:
		err := &ErrorUnsupportedEngineType{
			engineType: config.Type,
		}

		return err
	}

	db, err := sql.Open("postgres", config.CRDB.URI)
	if err != nil {
		return err
	}

	defer db.Close()

	return runMigrations(db)
}

// runMigrations runs all embedded migrations against the given database. Subsequent calls
// will have no effect. This function is safe to run across multiple goroutines.
func runMigrations(db *sql.DB) error {
	migrationWG.Wait()
	migrationWG.Add(1)

	defer migrationWG.Done()

	err := goose.Up(db, "migrations")

	return err
}
