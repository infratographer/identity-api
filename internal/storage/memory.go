package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cockroachdb/cockroach-go/v2/testserver"

	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

type memoryEngine struct {
	*memoryIssuerService
	crdb testserver.TestServer
}

func (eng *memoryEngine) Shutdown() {
	eng.crdb.Stop()
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

// GetByURI looks up the given issuer by URI, returning the issuer if one exists.
func (s *memoryIssuerService) GetByURI(ctx context.Context, uri string) (*v1.Issuer, error) {
	row := s.db.QueryRow(`SELECT id, name, uri, jwksuri, mappings FROM issuers WHERE uri = $1;`, uri)

	var iss v1.Issuer

	var mapping string

	err := row.Scan(&iss.ID, &iss.Name, &iss.URI, &iss.JWKSURI, &mapping)

	if errors.Is(err, sql.ErrNoRows) {
		err := v1.ErrorIssuerNotFound{
			URI: uri,
		}

		return nil, err
	} else if err != nil {
		return nil, err
	}

	c := v1.ClaimsMapping{}

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

func (s *memoryIssuerService) insertIssuer(iss v1.Issuer) error {
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
