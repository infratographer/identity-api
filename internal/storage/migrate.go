package storage

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var embedMigrations embed.FS

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

func runMigrations(db *sql.DB) error {
	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}

	return nil
}
