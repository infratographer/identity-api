package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cockroachdb/cockroach-go/v2/testserver"

	"go.infratographer.com/identity-manager-sts/internal/types"
)

type memoryEngine struct {
	*memoryIssuerService
	crdb testserver.TestServer
}

func (eng *memoryEngine) Shutdown() {
	eng.crdb.Stop()
}

func buildIssuerFromSeed(seed SeedIssuer) (types.Issuer, error) {
	claimMappings, err := types.NewClaimsMapping(seed.ClaimMappings)
	if err != nil {
		return types.Issuer{}, err
	}

	out := types.Issuer{
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
	db *sql.DB
}

// newMemoryEngine creates a new in-memory storage engine.
func newMemoryIssuerService(config Config) (*memoryIssuerService, error) {
	svc := &memoryIssuerService{db: config.db}

	err := svc.createTables()
	if err != nil {
		return nil, err
	}

	for _, seed := range config.SeedData.Issuers {
		iss, err := buildIssuerFromSeed(seed)
		if err != nil {
			return nil, err
		}

		err = svc.insertIssuer(iss)
		if err != nil {
			return nil, err
		}
	}

	return svc, nil
}

// CreateIssuer creates an issuer.
func (s *memoryIssuerService) CreateIssuer(ctx context.Context, iss types.Issuer) (*types.Issuer, error) {
	err := s.insertIssuer(iss)
	if err != nil {
		return nil, err
	}

	return &iss, nil
}

// GetIssuerByID gets an issuer by ID.
func (s *memoryIssuerService) GetIssuerByID(ctx context.Context, id string) (*types.Issuer, error) {
	row := s.db.QueryRow(`SELECT id, name, uri, jwksuri, mappings FROM issuers WHERE id = $1;`, id)

	iss, err := s.scanIssuer(row)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, types.ErrorIssuerNotFound
	case err != nil:
		return nil, err
	default:
		return iss, nil
	}
}

// GetByURI looks up the given issuer by URI, returning the issuer if one exists.
func (s *memoryIssuerService) GetIssuerByURI(ctx context.Context, uri string) (*types.Issuer, error) {
	row := s.db.QueryRow(`SELECT id, name, uri, jwksuri, mappings FROM issuers WHERE uri = $1;`, uri)

	iss, err := s.scanIssuer(row)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, types.ErrorIssuerNotFound
	case err != nil:
		return nil, err
	default:
		return iss, nil
	}
}

// DeleteIssuer deletes an issuer with the given ID.
func (s *memoryIssuerService) DeleteIssuer(ctx context.Context, id string) error {
	result, err := s.db.Exec(`DELETE FROM issuers WHERE id = $1;`, id)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return types.ErrorIssuerNotFound
	}

	return nil
}

func (s *memoryIssuerService) scanIssuer(row *sql.Row) (*types.Issuer, error) {
	var iss types.Issuer

	var mapping string

	err := row.Scan(&iss.ID, &iss.Name, &iss.URI, &iss.JWKSURI, &mapping)

	if err != nil {
		return nil, err
	}

	c := types.ClaimsMapping{}

	err = c.UnmarshalJSON([]byte(mapping))
	if err != nil {
		return nil, err
	}

	iss.ClaimMappings = c

	return &iss, nil
}

func (s *memoryIssuerService) createTables() error {
	stmt := `
        CREATE TABLE IF NOT EXISTS issuers (
            id       uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
            uri      STRING NOT NULL,
            name     STRING NOT NULL,
            jwksuri  STRING NOT NULL,
            mappings STRING
        );
        `
	_, err := s.db.Exec(stmt)

	return err
}

func (s *memoryIssuerService) insertIssuer(iss types.Issuer) error {
	q := `
        INSERT INTO issuers (
            id, name, uri, jwksuri, mappings
        ) VALUES
        ($1, $2, $3, $4, $5);
        `

	mappings, err := iss.ClaimMappings.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		q,
		iss.ID,
		iss.Name,
		iss.URI,
		iss.JWKSURI,
		string(mappings),
	)

	return err
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
