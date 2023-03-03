package storage

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
)

var _ types.OAuthClientManager = &oauthClientManager{}
var _ fosite.ClientManager = &oauthClientManager{}

func TestOAuthClientManager(t *testing.T) {
	t.Parallel()

	db, shutdown := testserver.NewDBForTest(t)

	err := runMigrations(db)
	if err != nil {
		shutdown()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		shutdown()
	})

	tenantID := "56a95c1b-33f8-4def-8b6d-ca9fe6976170"
	issuer := types.Issuer{
		TenantID: tenantID,
		ID:       "e495a393-ae79-4a02-a78d-9798c7d9d252",
		Name:     "Example",
		URI:      "https://example.com/",
		JWKSURI:  "https://example.com/.well-known/jwks.json",
	}

	seedIssuers := []SeedIssuer{
		{
			TenantID: tenantID,
			ID:       issuer.ID,
			Name:     issuer.Name,
			URI:      issuer.URI,
			JWKSURI:  issuer.JWKSURI,
		},
	}

	issSvc, err := newIssuerService(db)
	assert.NoError(t, err)

	assert.NoError(t, issSvc.seedDatabase(context.Background(), seedIssuers))

	oauthClientStore, err := newOAuthClientManager(db)
	assert.NoError(t, err)

	defaultClient := types.OAuthClient{
		TenantID: tenantID,
		Name:     "my-client",
		Secret:   "foobar",
		Audience: []string{"aud1", "aud2"},
	}

	seedCtx, err := beginTxContext(context.Background(), db)
	require.NoError(t, err)

	defaultClient, err = oauthClientStore.CreateOAuthClient(seedCtx, defaultClient)
	require.NoError(t, err)
	require.NoError(t, commitContextTx(seedCtx))

	setupWithTx := func(ctx context.Context) context.Context {
		ctx, err := beginTxContext(ctx, db)
		if err != nil {
			t.Fatal("failed to start transaction")
		}

		return ctx
	}

	cleanupWithTx := func(ctx context.Context) {
		err := rollbackContextTx(ctx)
		if err != nil {
			t.Fatal("failed to roll back transaction")
		}
	}

	t.Run("LookupClientByID", func(t *testing.T) {
		t.Parallel()

		runFn := func(ctx context.Context, input string) testingx.TestResult[types.OAuthClient] {
			res, err := oauthClientStore.LookupOAuthClientByID(ctx, input)
			return testingx.TestResult[types.OAuthClient]{
				Success: res,
				Err:     err,
			}
		}

		testCases := []testingx.TestCase[string, types.OAuthClient]{
			{
				Name:  "NotFoundWithoutTx",
				Input: uuid.NewString(),
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[types.OAuthClient]) {
					assert.ErrorIs(t, res.Err, types.ErrOAuthClientNotFound)
				},
			},
			{
				Name:    "NotFoundWithTx",
				Input:   uuid.NewString(),
				SetupFn: setupWithTx,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[types.OAuthClient]) {
					assert.ErrorIs(t, res.Err, types.ErrOAuthClientNotFound)
				},
				CleanupFn: cleanupWithTx,
			},
			{
				Name:    "FoundWithTx",
				Input:   defaultClient.ID,
				SetupFn: setupWithTx,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[types.OAuthClient]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, defaultClient, res.Success)
					assert.NotEqual(t, "foobar", res.Success.Secret)
				},
				CleanupFn: cleanupWithTx,
			},
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("CreateOAuthClient", func(t *testing.T) {
		t.Parallel()

		runFn := func(ctx context.Context, input types.OAuthClient) testingx.TestResult[types.OAuthClient] {
			out, err := oauthClientStore.CreateOAuthClient(ctx, input)
			return testingx.TestResult[types.OAuthClient]{
				Success: out,
				Err:     err,
			}
		}

		secret := "superdupersecret"
		testCases := []testingx.TestCase[types.OAuthClient, types.OAuthClient]{
			{
				Name: "Success",
				Input: types.OAuthClient{
					TenantID: tenantID,
					Name:     "newclient",
					Secret:   secret,
					Audience: []string{"abc", "def", "ghi"},
				},
				SetupFn:   setupWithTx,
				CleanupFn: cleanupWithTx,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[types.OAuthClient]) {
					assert.NoError(t, res.Err)
					client := res.Success
					assert.NotEqual(t, secret, client.Secret)
					assert.Equal(t, tenantID, client.TenantID)
					assert.Equal(t, "newclient", client.Name)
					assert.Equal(t, []string{"abc", "def", "ghi"}, client.Audience)
				},
			},
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("DeleteOAuthClient", func(t *testing.T) {
		t.Parallel()

		runFn := func(ctx context.Context, input string) testingx.TestResult[any] {
			err := oauthClientStore.DeleteOAuthClient(ctx, input)
			return testingx.TestResult[any]{
				Err: err,
			}
		}

		testCases := []testingx.TestCase[string, any]{
			{
				Name:      "ValidClientWithTx",
				Input:     defaultClient.ID,
				SetupFn:   setupWithTx,
				CleanupFn: cleanupWithTx,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					assert.NoError(t, res.Err)
					_, err := oauthClientStore.LookupOAuthClientByID(ctx, defaultClient.ID)
					assert.ErrorIs(t, err, types.ErrOAuthClientNotFound)
				},
			},
			{
				Name:  "ValidClientWithoutTx",
				Input: defaultClient.ID,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					assert.ErrorIs(t, res.Err, ErrorMissingContextTx)
					c, err := oauthClientStore.LookupOAuthClientByID(ctx, defaultClient.ID)
					assert.NoError(t, err)
					assert.NotEmpty(t, c)
				},
			},

			{
				Name:      "NotFound",
				Input:     uuid.NewString(),
				SetupFn:   setupWithTx,
				CleanupFn: cleanupWithTx,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					assert.NoError(t, res.Err)
				},
			},
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})
}
