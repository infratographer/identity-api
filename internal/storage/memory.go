package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	*memoryUserInfoService
	crdb testserver.TestServer
	db   *sql.DB
}

func (eng *memoryEngine) Shutdown() {
	eng.crdb.Stop()
}

func (eng *memoryEngine) BeginContext(ctx context.Context) (context.Context, error) {
	return beginTxContext(ctx, eng.db)
}

func (eng *memoryEngine) CommitContext(ctx context.Context) error {
	return commitContextTx(ctx)
}

func (eng *memoryEngine) RollbackContext(ctx context.Context) error {
	return rollbackContextTx(ctx)
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

	ctx, err := beginTxContext(context.Background(), config.db)
	if err != nil {
		return nil, err
	}

	for _, seed := range config.SeedData.Issuers {
		iss, err := buildIssuerFromSeed(seed)
		if err != nil {
			return nil, err
		}

		err = svc.insertIssuer(ctx, iss)
		if err != nil {
			return nil, err
		}
	}

	err = commitContextTx(ctx)
	if err != nil {
		if err := rollbackContextTx(ctx); err != nil {
			return nil, err
		}

		return nil, err
	}

	return svc, nil
}

// CreateIssuer creates an issuer.
func (s *memoryIssuerService) CreateIssuer(ctx context.Context, iss types.Issuer) (*types.Issuer, error) {
	err := s.insertIssuer(ctx, iss)
	if err != nil {
		return nil, err
	}

	return &iss, nil
}

// GetIssuerByID gets an issuer by ID. This function will use a transaction in the context if one
// exists.
func (s *memoryIssuerService) GetIssuerByID(ctx context.Context, id string) (*types.Issuer, error) {
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

// GetByURI looks up the given issuer by URI, returning the issuer if one exists. This function will
// use a transaction in the context if one exists.
func (s *memoryIssuerService) GetIssuerByURI(ctx context.Context, uri string) (*types.Issuer, error) {
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
func (s *memoryIssuerService) UpdateIssuer(ctx context.Context, id string, update types.IssuerUpdate) (*types.Issuer, error) {
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
func (s *memoryIssuerService) DeleteIssuer(ctx context.Context, id string) error {
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
            uri       STRING NOT NULL UNIQUE,
            name      STRING NOT NULL,
            jwksuri   STRING NOT NULL,
            mappings  STRING
        );
        `
	_, err := s.db.Exec(stmt)

	return err
}

func (s *memoryIssuerService) insertIssuer(ctx context.Context, iss types.Issuer) error {
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

type memoryUserInfoService struct {
	db         *sql.DB
	httpClient *http.Client
}

type userInfoServiceOpt func(*memoryUserInfoService)

func newUserInfoService(config Config, opts ...userInfoServiceOpt) (*memoryUserInfoService, error) {
	s := &memoryUserInfoService{
		db:         config.db,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(s)
	}

	err := s.createTables()

	return s, err
}

// WithHTTPClient allows configuring the HTTP client used by
// memoryUserInfoService to call out to userinfo endpoints.
func WithHTTPClient(client *http.Client) func(svc *memoryUserInfoService) {
	return func(svc *memoryUserInfoService) {
		svc.httpClient = client
	}
}

func (s *memoryUserInfoService) createTables() error {
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS user_info (
            id    UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
            name  STRING,
            email STRING,
            sub   STRING NOT NULL,
            iss_id   UUID NOT NULL REFERENCES issuers(id),
            UNIQUE (iss_id, sub)
        )`)

	return err
}

// LookupUserInfoByClaims fetches UserInfo from the store.
// This does not make an HTTP call with the subject token, so for this
// data to be available, the data must have already be fetched and
// stored.
func (s memoryUserInfoService) LookupUserInfoByClaims(ctx context.Context, iss, sub string) (*types.UserInfo, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT ui.name, ui.email, ui.sub, i.uri FROM user_info ui
        JOIN issuers i ON
           ui.iss_id = i.id
        WHERE
           i.uri = $1 AND ui.sub = $2
        `, iss, sub)

	var ui types.UserInfo

	err := row.Scan(&ui.Name, &ui.Email, &ui.Subject, &ui.Issuer)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrUserInfoNotFound
	}

	return &ui, err
}

// StoreUserInfo is used to store user information by issuer and
// subject pairs. UserInfo is unique to issuer/subject pairs.
func (s memoryUserInfoService) StoreUserInfo(ctx context.Context, userInfo types.UserInfo) error {
	row := s.db.QueryRowContext(ctx, `
        SELECT id FROM issuers WHERE uri = $1
        `, userInfo.Issuer)

	var issuerID string

	err := row.Scan(&issuerID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
        INSERT INTO user_info (name, email, sub, iss_id) VALUES (
            $1, $2, $3, $4
	)`, userInfo.Name, userInfo.Email, userInfo.Subject, issuerID)

	return err
}

// FetchUserInfoFromIssuer uses the subject access token to retrieve
// information from the OIDC /userinfo endpoint.
func (s memoryUserInfoService) FetchUserInfoFromIssuer(ctx context.Context, iss, rawToken string) (*types.UserInfo, error) {
	endpoint, err := url.JoinPath(iss, "userinfo")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", rawToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected response code %d from request: %w",
			resp.StatusCode,
			types.ErrFetchUserInfo,
		)
	}

	var ui types.UserInfo

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ui)
	if err != nil {
		return nil, err
	}

	return &ui, nil
}
