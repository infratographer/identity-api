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

	"github.com/google/uuid"

	"go.infratographer.com/identity-api/internal/types"
)

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

type userInfoService struct {
	db         *sql.DB
	httpClient *http.Client
}

type userInfoServiceOpt func(*userInfoService)

func newUserInfoService(db *sql.DB, opts ...userInfoServiceOpt) (*userInfoService, error) {
	s := &userInfoService{
		db:         db,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// WithHTTPClient allows configuring the HTTP client used by
// userInfoService to call out to userinfo endpoints.
func WithHTTPClient(client *http.Client) func(svc *userInfoService) {
	return func(svc *userInfoService) {
		svc.httpClient = client
	}
}

// LookupUserInfoByClaims fetches UserInfo from the store.
// This does not make an HTTP call with the subject token, so for this
// data to be available, the data must have already be fetched and
// stored.
func (s userInfoService) LookupUserInfoByClaims(ctx context.Context, iss, sub string) (*types.UserInfo, error) {
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

func (s userInfoService) LookupUserInfoByID(ctx context.Context, id string) (*types.UserInfo, error) {
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
func (s userInfoService) StoreUserInfo(ctx context.Context, userInfo types.UserInfo) (*types.UserInfo, error) {
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
func (s userInfoService) FetchUserInfoFromIssuer(ctx context.Context, iss, rawToken string) (*types.UserInfo, error) {
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
