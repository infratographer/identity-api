package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach-go/v2/testserver"

	"go.infratographer.com/identity-manager-sts/internal/types"
)

const (
	issuerColTenantID = "tenant_id"
	issuerColID       = "id"
	issuerColName     = "name"
	issuerColURI      = "uri"
	issuerColJWKSURI  = "jwksuri"
	issuerColMappings = "mappings"
)

var (
	issuerColumns = []string{
		issuerColTenantID,
		issuerColID,
		issuerColName,
		issuerColURI,
		issuerColJWKSURI,
		issuerColMappings,
	}
	issuerColumnsStr = strings.Join(issuerColumns, ", ")
)

type colBinding struct {
	column string
	value  any
}

func colBindingsToParams(bindings []colBinding) (string, []any) {
	bindingStrs := make([]string, len(bindings))
	args := make([]any, len(bindings))

	for i, binding := range bindings {
		bindingStr := fmt.Sprintf("%s = $%d", binding.column, i+1)
		bindingStrs[i] = bindingStr
		args[i] = binding.value
	}

	bindingsStr := strings.Join(bindingStrs, ", ")

	return bindingsStr, args
}

func bindIfNotNil[T any](bindings []colBinding, column string, value *T) []colBinding {
	if value != nil {
		binding := colBinding{
			column: column,
			value:  *value,
		}

		return append(bindings, binding)
	}

	return bindings
}

func issuerUpdateToColBindings(update types.IssuerUpdate) ([]colBinding, error) {
	var bindings []colBinding

	bindings = bindIfNotNil(bindings, issuerColName, update.Name)
	bindings = bindIfNotNil(bindings, issuerColURI, update.URI)
	bindings = bindIfNotNil(bindings, issuerColJWKSURI, update.JWKSURI)

	if update.ClaimMappings != nil {
		mappingRepr, err := update.ClaimMappings.MarshalJSON()
		if err != nil {
			return nil, err
		}

		mappingStr := string(mappingRepr)

		bindings = bindIfNotNil(bindings, issuerColMappings, &mappingStr)
	}

	return bindings, nil
}

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
		TenantID:      seed.TenantID,
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
	query := fmt.Sprintf("SELECT %s FROM issuers WHERE id = $1", issuerColumnsStr)
	row := s.db.QueryRow(query, id)

	return s.scanIssuer(row)
}

// GetByURI looks up the given issuer by URI, returning the issuer if one exists.
func (s *memoryIssuerService) GetIssuerByURI(ctx context.Context, uri string) (*types.Issuer, error) {
	query := fmt.Sprintf("SELECT %s FROM issuers WHERE uri = $1", issuerColumnsStr)
	row := s.db.QueryRow(query, uri)

	return s.scanIssuer(row)
}

// UpdateIssuer updates an issuer with the given values.
func (s *memoryIssuerService) UpdateIssuer(ctx context.Context, id string, update types.IssuerUpdate) (*types.Issuer, error) {
	bindings, err := issuerUpdateToColBindings(update)
	if err != nil {
		return nil, err
	}

	params, args := colBindingsToParams(bindings)

	query := fmt.Sprintf("UPDATE issuers SET %s WHERE id = $%d RETURNING %s", params, len(args)+1, issuerColumnsStr)

	args = append(args, id)

	row := s.db.QueryRow(query, args...)

	return s.scanIssuer(row)
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

	var mapping sql.NullString

	err := row.Scan(&iss.TenantID, &iss.ID, &iss.Name, &iss.URI, &iss.JWKSURI, &mapping)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, types.ErrorIssuerNotFound
	case err != nil:
		return nil, err
	default:
	}

	c := types.ClaimsMapping{}

	if mapping.Valid {
		err = c.UnmarshalJSON([]byte(mapping.String))
		if err != nil {
			return nil, err
		}

		iss.ClaimMappings = c
	}

	return &iss, nil
}

func (s *memoryIssuerService) createTables() error {
	stmt := `
        CREATE TABLE IF NOT EXISTS issuers (
            id        uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
            tenant_id uuid NOT NULL,
            uri       STRING NOT NULL,
            name      STRING NOT NULL,
            jwksuri   STRING NOT NULL,
            mappings  STRING
        );
        `
	_, err := s.db.Exec(stmt)

	return err
}

func (s *memoryIssuerService) insertIssuer(iss types.Issuer) error {
	q := `
        INSERT INTO issuers (
            %s
        ) VALUES
        ($1, $2, $3, $4, $5, $6);
        `

	q = fmt.Sprintf(q, issuerColumnsStr)

	mappings, err := iss.ClaimMappings.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		q,
		iss.TenantID,
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
