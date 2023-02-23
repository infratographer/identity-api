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

// RunMigrations runs all migrations against the given SQL database.
func RunMigrations(db *sql.DB) error {
	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}

	return nil
}
