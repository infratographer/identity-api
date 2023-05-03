package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ory/fosite"
	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/x/gidx"
)

var oauthClientCols = struct {
	ID       string
	TenantID string
	Name     string
	Secret   string
	Audience string
}{
	ID:       "id",
	TenantID: "tenant_id",
	Name:     "name",
	Secret:   "secret",
	Audience: "audience",
}

var (
	oauthClientColumns = []string{
		oauthClientCols.ID,
		oauthClientCols.TenantID,
		oauthClientCols.Name,
		oauthClientCols.Secret,
		oauthClientCols.Audience,
	}
	oauthClientColumnsStr = strings.Join(oauthClientColumns, ", ")
)

type oauthClientManager struct {
	db     *sql.DB
	hasher fosite.Hasher
}

// ClientAssertionJWTValid implements fosite.ClientManager
func (*oauthClientManager) ClientAssertionJWTValid(_ context.Context, _ string) error {
	panic("unimplemented")
}

// GetClient implements fosite.ClientManager
func (s *oauthClientManager) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	clientID, err := gidx.Parse(id)
	if err != nil {
		return nil, err
	}

	return s.LookupOAuthClientByID(ctx, clientID)
}

// SetClientAssertionJWT implements fosite.ClientManager
func (*oauthClientManager) SetClientAssertionJWT(_ context.Context, _ string, _ time.Time) error {
	panic("unimplemented")
}

func newOAuthClientManager(db *sql.DB) (*oauthClientManager, error) {
	return &oauthClientManager{
		db: db,
		hasher: &fosite.BCrypt{
			Config: &fosite.Config{
				HashCost: fosite.DefaultBCryptWorkFactor,
			},
		},
	}, nil
}

// CreateOAuthClient creates an OAuth client in the database.
func (s *oauthClientManager) CreateOAuthClient(ctx context.Context, client types.OAuthClient) (types.OAuthClient, error) {
	var emptyModel types.OAuthClient

	tx, err := getContextTx(ctx)
	if err != nil {
		return emptyModel, err
	}

	q := `
        INSERT INTO oauth_clients (
           %s
        ) VALUES
        ($1, $2, $3, $4, $5) RETURNING id;
       `
	q = fmt.Sprintf(q, oauthClientColumnsStr)

	hashedSecret, err := s.hasher.Hash(ctx, []byte(client.Secret))
	if err != nil {
		return emptyModel, err
	}

	clientID, err := gidx.NewID(types.IdentityClientIDPrefix)
	if err != nil {
		return emptyModel, err
	}

	client.Secret = string(hashedSecret)

	row := tx.QueryRowContext(
		ctx,
		q,
		clientID,
		client.TenantID,
		client.Name,
		client.Secret,
		strings.Join(client.Audience, " "),
	)

	err = row.Scan(&client.ID)
	if err != nil {
		return emptyModel, err
	}

	return client, nil
}

// DeleteOAuthClient removes the OAuth client from the store.
func (*oauthClientManager) DeleteOAuthClient(ctx context.Context, clientID gidx.PrefixedID) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM oauth_clients WHERE id = $1;`, clientID)

	return err
}

// LookupOAuthClientByID fetches an OAuth client from the store.
func (s *oauthClientManager) LookupOAuthClientByID(ctx context.Context, clientID gidx.PrefixedID) (types.OAuthClient, error) {
	if clientID == "" {
		return types.OAuthClient{}, types.ErrOAuthClientNotFound
	}

	q := fmt.Sprintf(`SELECT %s FROM oauth_clients WHERE id = $1`, oauthClientColumnsStr)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, q, clientID)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, q, clientID)
	default:
		return types.OAuthClient{}, err
	}

	var model types.OAuthClient

	var aud string

	err = row.Scan(
		&model.ID,
		&model.TenantID,
		&model.Name,
		&model.Secret,
		&aud,
	)

	switch err {
	case nil:
	case sql.ErrNoRows:
		return types.OAuthClient{}, types.ErrOAuthClientNotFound
	default:
		return types.OAuthClient{}, err
	}

	model.Audience = strings.Fields(aud)

	return model, nil
}
