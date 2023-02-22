package storage

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
)

func compareIssuers(t *testing.T, exp types.Issuer, obs types.Issuer) {
	// Clear out mappings, since some of the AST can change between marshaling/unmarshaling
	expMappings, err := exp.ClaimMappings.Repr()
	if err != nil {
		t.Fatal(err)
	}

	obsMappings, err := obs.ClaimMappings.Repr()
	if err != nil {
		t.Fatal(err)
	}

	exp.ClaimMappings = nil
	obs.ClaimMappings = nil

	assert.Equal(t, exp, obs)
	assert.Equal(t, expMappings, obsMappings)
}

func TestIssuerService(t *testing.T) {
	t.Parallel()

	db, shutdown := testserver.NewDBForTest(t)

	err := RunMigrations(db)
	if err != nil {
		shutdown()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		shutdown()
	})

	mappingStrs := map[string]string{
		"foo": "123",
	}

	mappings, err := types.NewClaimsMapping(mappingStrs)
	if err != nil {
		panic(err)
	}

	tenantID := "56a95c1b-33f8-4def-8b6d-ca9fe6976170"
	issuer := types.Issuer{
		TenantID:      tenantID,
		ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
		Name:          "Example",
		URI:           "https://example.com/",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: mappings,
	}

	config := Config{
		db: db,
		SeedData: SeedData{
			Issuers: []SeedIssuer{
				{
					TenantID:      tenantID,
					ID:            issuer.ID,
					Name:          issuer.Name,
					URI:           issuer.URI,
					JWKSURI:       issuer.JWKSURI,
					ClaimMappings: mappingStrs,
				},
			},
		},
	}

	issSvc, err := newIssuerService(config)
	assert.Nil(t, err)

	t.Run("CreateIssuer", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            "6b0117f8-29e4-49fa-841e-63c52aa27d96",
			Name:          "Good issuer",
			URI:           "https://issuer-a27d96.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		testCases := []testingx.TestCase[types.Issuer, *types.Issuer]{
			{
				Name:  "Success",
				Input: issuer,
				SetupFn: func(ctx context.Context) context.Context {
					txCtx, err := beginTxContext(ctx, db)
					if !assert.NoError(t, err) {
						assert.FailNow(t, "setup failed")
					}

					return txCtx
				},
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
				CleanupFn: func(ctx context.Context) {
					err := rollbackContextTx(ctx)
					assert.NoError(t, err)
				},
			},
		}

		runFn := func(ctx context.Context, input types.Issuer) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.CreateIssuer(ctx, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetIssuerByURI", func(t *testing.T) {
		t.Parallel()

		testCases := []testingx.TestCase[string, *types.Issuer]{
			{
				Name:  "NotFound",
				Input: "https://evil.biz/",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
			{
				Name: "UsingTx",
				SetupFn: func(ctx context.Context) context.Context {
					txCtx, err := beginTxContext(ctx, db)
					if !assert.NoError(t, err) {
						assert.FailNow(t, "setup failed")
					}

					return txCtx
				},
				Input: "https://example.com/",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
				CleanupFn: func(ctx context.Context) {
					err := rollbackContextTx(ctx)
					assert.NoError(t, err)
				},
			},
			{
				Name:  "UsingDB",
				Input: "https://example.com/",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.GetIssuerByURI(ctx, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetIssuerByID", func(t *testing.T) {
		t.Parallel()

		testCases := []testingx.TestCase[string, *types.Issuer]{
			{
				Name:  "NotFound",
				Input: "00000000-0000-0000-0000-000000000000",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
			{
				Name:  "UsingTx",
				Input: issuer.ID,
				SetupFn: func(ctx context.Context) context.Context {
					txCtx, err := beginTxContext(ctx, db)
					if !assert.NoError(t, err) {
						assert.FailNow(t, "setup failed")
					}

					return txCtx
				},
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
				CleanupFn: func(ctx context.Context) {
					err := rollbackContextTx(ctx)
					assert.NoError(t, err)
				},
			},

			{
				Name:  "UsingDB",
				Input: issuer.ID,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.GetIssuerByID(ctx, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("UpdateIssuer", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            "b9ae2e16-11c0-49e4-8d9b-1d6698bba1a3",
			Name:          "Good issuer",
			URI:           "https://issuer-bba1a3.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		newName := "Better issuer"
		newURI := "https://issuer.info/better/"
		newJWKSURI := "https://issuer.info/better/jwks.json"
		newMapping := types.ClaimsMapping{}

		fullUpdate := types.IssuerUpdate{
			Name:          &newName,
			URI:           &newURI,
			JWKSURI:       &newJWKSURI,
			ClaimMappings: newMapping,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := beginTxContext(ctx, db)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = issSvc.CreateIssuer(ctx, issuer)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := rollbackContextTx(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[types.IssuerUpdate, *types.Issuer]{
			{
				Name:    "Full",
				Input:   fullUpdate,
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					exp := issuer
					exp.Name = newName
					exp.URI = newURI
					exp.JWKSURI = newJWKSURI
					exp.ClaimMappings = newMapping

					if assert.NoError(t, res.Err) {
						compareIssuers(t, exp, *res.Success)
					}
				},
				CleanupFn: cleanupFn,
			},
		}

		runFn := func(ctx context.Context, input types.IssuerUpdate) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.UpdateIssuer(ctx, issuer.ID, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("DeleteIssuer", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            "ace77968-03b0-4f1b-b3f0-b214daf4ac18",
			Name:          "Good issuer",
			URI:           "https://issuer-f4ac18.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := beginTxContext(ctx, db)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = issSvc.CreateIssuer(ctx, issuer)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := rollbackContextTx(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[string, any]{
			{
				Name:    "Success",
				Input:   issuer.ID,
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					if assert.NoError(t, res.Err) {
						_, err := issSvc.GetIssuerByID(ctx, issuer.ID)
						assert.ErrorIs(t, types.ErrorIssuerNotFound, err)
					}
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "NotFound",
				Input:   "00000000-0000-0000-0000-000000000000",
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
				CleanupFn: cleanupFn,
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[any] {
			err = issSvc.DeleteIssuer(ctx, input)

			result := testingx.TestResult[any]{
				Success: nil,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})
}
