package storage

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

type memoryEngine struct {
	*memoryIssuerService
}

func buildIssuerFromSeed(seed SeedIssuer) (v1.Issuer, error) {
	claimMappings, err := v1.BuildClaimsMappingFromMap(seed.ClaimMappings)
	if err != nil {
		return v1.Issuer{}, err
	}

	out := v1.Issuer{
		ID:            seed.ID,
		Name:          seed.Name,
		URI:           seed.URI,
		JWKSURI:       seed.JWKSURI,
		ClaimMappings: claimMappings,
	}

	return out, nil
}

// memoryIssuerService represents an in-memory issuer service.
type memoryIssuerService struct {
	db      *sql.DB
	issuers map[string]v1.Issuer
}

// newMemoryEngine creates a new in-memory storage engine.
func newMemoryIssuerService(config Config) (*memoryIssuerService, error) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return nil, err
	}

	createTables(db)
	for _, seed := range config.SeedData.Issuers {
		iss, err := buildIssuerFromSeed(seed)
		if err != nil {
			return nil, err
		}

		err = insertIssuer(db, iss)
		if err != nil {
			return nil, err
		}
	}

	out := &memoryIssuerService{
		db: db,
	}

	return out, nil
}

// GetByURI looks up the given issuer by URI, returning the issuer if one exists.
func (s *memoryIssuerService) GetByURI(ctx context.Context, uri string) (*v1.Issuer, error) {
	row := s.db.QueryRow(`SELECT id, name, uri, jwksuri, mappings FROM issuers WHERE uri = ?;`, uri)

	var iss v1.Issuer
	var mapping string
	err := row.Scan(&iss.ID, &iss.Name, &iss.URI, &iss.JWKSURI, &mapping)

	if err == sql.ErrNoRows {
		err := v1.ErrorIssuerNotFound{
			URI: uri,
		}

		return nil, err
	} else if err != nil {
		return nil, err
	}

	c := v1.ClaimsMapping{}
	c.UnmarshalJSON([]byte(mapping))
	iss.ClaimMappings = c

	return &iss, nil
}

func createTables(db *sql.DB) error {
	stmt := `
        CREATE TABLE IF NOT EXISTS issuers (
            uri      TEXT NOT NULL PRIMARY KEY,
            id       TEXT,
            name     TEXT,
            jwksuri  TEXT,
            mappings TEXT
        );
        `
	_, err := db.Exec(stmt)
	return err
}

func insertIssuer(db *sql.DB, iss v1.Issuer) error {
	q := `
        INSERT INTO issuers (
            id, name, uri, jwksuri, mappings
        ) VALUES
        (?, ?, ?, ?, ?);
        `

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	mappings, err := iss.ClaimMappings.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = stmt.Exec(
		iss.ID,
		iss.Name,
		iss.URI,
		iss.JWKSURI,
		string(mappings),
	)

	return err
}
