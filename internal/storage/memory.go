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
	"github.com/google/uuid"

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

var userInfoCols = struct {
	ID       string
	Name     string
	Email    string
	Subject  string
	IssuerID string
}{
	ID:       "id",
	Name:     "name",
	Email:    "email",
	Subject:  "sub",
	IssuerID: "iss_id",
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

	bindings = bindIfNotNil(bindings, issuerCols.Name, update.Name)
	bindings = bindIfNotNil(bindings, issuerCols.URI, update.URI)
	bindings = bindIfNotNil(bindings, issuerCols.JWKSURI, update.JWKSURI)

	if update.ClaimMappings != nil {
		mappingRepr, err := update.ClaimMappings.MarshalJSON()
		if err != nil {
			return nil, err
		}

		mappingStr := string(mappingRepr)

		bindings = bindIfNotNil(bindings, issuerCols.Mappings, &mappingStr)
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
            name  STRING NOT NULL,
            email STRING NOT NULL,
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
	selectCols := withQualifier([]string{
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
	}, "ui")

	selectCols = append(selectCols, "i."+issuerCols.URI)

	selects := strings.Join(selectCols, ",")

	stmt := fmt.Sprintf(`
	SELECT %[1]s FROM user_info ui
        JOIN issuers i ON ui.%[2]s = i.%[3]s
        WHERE i.%[4]s = $1 and ui.%[5]s = $2`,
		selects,
		userInfoCols.IssuerID,
		issuerCols.ID,
		issuerCols.URI,
		userInfoCols.Subject,
	)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, stmt, iss, sub)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, stmt, iss, sub)
	default:
		return nil, err
	}

	var ui types.UserInfo

	err = row.Scan(&ui.Name, &ui.Email, &ui.Subject, &ui.Issuer)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrUserInfoNotFound
	}

	return &ui, err
}

func (s memoryUserInfoService) LookupUserInfoByID(ctx context.Context, id string) (*types.UserInfo, error) {
	selectCols := withQualifier([]string{
		userInfoCols.ID,
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
	}, "ui")

	selectCols = append(selectCols, "i."+issuerCols.URI)

	selects := strings.Join(selectCols, ",")

	stmt := fmt.Sprintf(`
        SELECT %[1]s FROM user_info ui
        JOIN issuers i ON ui.%[2]s = i.%[3]s
        WHERE ui.%[4]s = $1
        `, selects, userInfoCols.IssuerID, issuerCols.ID, userInfoCols.ID)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, stmt, id)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, stmt, id)
	default:
		return nil, err
	}

	var ui types.UserInfo

	err = row.Scan(&ui.ID, &ui.Name, &ui.Email, &ui.Subject, &ui.Issuer)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrUserInfoNotFound
	}

	return &ui, err
}

// StoreUserInfo is used to store user information by issuer and
// subject pairs. UserInfo is unique to issuer/subject pairs.
func (s memoryUserInfoService) StoreUserInfo(ctx context.Context, userInfo types.UserInfo) (*types.UserInfo, error) {
	if len(userInfo.Issuer) == 0 {
		return nil, fmt.Errorf("%w: issuer is empty", types.ErrInvalidUserInfo)
	}

	if len(userInfo.Subject) == 0 {
		return nil, fmt.Errorf("%w: subject is empty", types.ErrInvalidUserInfo)
	}

	tx, err := getContextTx(ctx)
	if err != nil {
		return nil, err
	}

	row := tx.QueryRowContext(ctx, `
        SELECT id FROM issuers WHERE uri = $1
        `, userInfo.Issuer)

	var issuerID string

	err = row.Scan(&issuerID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, types.ErrorIssuerNotFound
	default:
		return nil, err
	}

	insertCols := strings.Join([]string{
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
		userInfoCols.IssuerID,
	}, ",")

	q := fmt.Sprintf(`INSERT INTO user_info (%[1]s) VALUES (
            $1, $2, $3, $4
	) ON CONFLICT (%[2]s, %[3]s)
        DO UPDATE SET %[2]s = excluded.%[2]s, %[3]s = excluded.%[3]s
        RETURNING id`,
		insertCols,
		userInfoCols.Subject,
		userInfoCols.IssuerID,
	)

	row = tx.QueryRowContext(ctx, q,
		userInfo.Name, userInfo.Email, userInfo.Subject, issuerID,
	)

	var userID string

	err = row.Scan(&userID)
	if err != nil {
		return nil, err
	}

	userInfo.ID = uuid.MustParse(userID)

	return &userInfo, err
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

	if ui.Issuer == "" {
		ui.Issuer = iss
	}

	return &ui, nil
}

// withQualifier adds a qualifier to a column
// e.g. withQualifier([]string{"name"}, "ui") = []string{"ui.name"}
func withQualifier(items []string, qualifier string) []string {
	out := make([]string, len(items))
	for i, el := range items {
		out[i] = fmt.Sprintf("%s.%s", qualifier, el)
	}

	return out
}
