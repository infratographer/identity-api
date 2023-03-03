package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.infratographer.com/identity-api/internal/types"
)

var issuerCols = struct {
	TenantID string
	ID       string
	Name     string
	URI      string
	JWKSURI  string
	Mappings string
}{
	TenantID: "tenant_id",
	ID:       "id",
	Name:     "name",
	URI:      "uri",
	JWKSURI:  "jwksuri",
	Mappings: "mappings",
}

var (
	issuerColumns = []string{
		issuerCols.TenantID,
		issuerCols.ID,
		issuerCols.Name,
		issuerCols.URI,
		issuerCols.JWKSURI,
		issuerCols.Mappings,
	}
	issuerColumnsStr = strings.Join(issuerColumns, ", ")
)

// issuerService represents a SQL-backed issuer service.
type issuerService struct {
	db *sql.DB
}

func newIssuerService(db *sql.DB) (*issuerService, error) {
	svc := &issuerService{
		db: db,
	}

	return svc, nil
}

func (s *issuerService) seedDatabase(ctx context.Context, issuers []SeedIssuer) error {
	ctx, err := beginTxContext(ctx, s.db)
	if err != nil {
		return err
	}

	for _, seed := range issuers {
		iss, err := buildIssuerFromSeed(seed)
		if err != nil {
			return err
		}

		err = s.insertIssuer(ctx, iss)
		if err != nil {
			return err
		}
	}

	err = commitContextTx(ctx)
	if err != nil {
		if err := rollbackContextTx(ctx); err != nil {
			return err
		}

		return err
	}

	return nil
}

// CreateIssuer creates an issuer.
func (s *issuerService) CreateIssuer(ctx context.Context, iss types.Issuer) (*types.Issuer, error) {
	err := s.insertIssuer(ctx, iss)
	if err != nil {
		return nil, err
	}

	return &iss, nil
}

// GetIssuerByID gets an issuer by ID. This function will use a transaction in the context if one
// exists.
func (s *issuerService) GetIssuerByID(ctx context.Context, id string) (*types.Issuer, error) {
	query := fmt.Sprintf("SELECT %s FROM issuers WHERE id = $1", issuerColumnsStr)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, query, id)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, query, id)
	default:
		return nil, err
	}

	return s.scanIssuer(row)
}

// GetIssuerByURI looks up the given issuer by URI, returning the issuer if one exists. This function will
// use a transaction in the context if one exists.
func (s *issuerService) GetIssuerByURI(ctx context.Context, uri string) (*types.Issuer, error) {
	query := fmt.Sprintf("SELECT %s FROM issuers WHERE uri = $1", issuerColumnsStr)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, query, uri)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, query, uri)
	default:
		return nil, err
	}

	return s.scanIssuer(row)
}

// UpdateIssuer updates an issuer with the given values.
func (s *issuerService) UpdateIssuer(ctx context.Context, id string, update types.IssuerUpdate) (*types.Issuer, error) {
	tx, err := getContextTx(ctx)
	if err != nil {
		return nil, err
	}

	bindings, err := issuerUpdateToColBindings(update)
	if err != nil {
		return nil, err
	}

	params, args := colBindingsToParams(bindings)

	query := fmt.Sprintf("UPDATE issuers SET %s WHERE id = $%d RETURNING %s", params, len(args)+1, issuerColumnsStr)

	args = append(args, id)

	row := tx.QueryRowContext(ctx, query, args...)

	return s.scanIssuer(row)
}

// DeleteIssuer deletes an issuer with the given ID.
func (s *issuerService) DeleteIssuer(ctx context.Context, id string) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM issuers WHERE id = $1;`, id)

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

func (s *issuerService) scanIssuer(row *sql.Row) (*types.Issuer, error) {
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

func (s *issuerService) insertIssuer(ctx context.Context, iss types.Issuer) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

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

	_, err = tx.ExecContext(
		ctx,
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
