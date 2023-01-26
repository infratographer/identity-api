package storage

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-manager-sts/internal/testingx"
	"go.infratographer.com/identity-manager-sts/internal/types"
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

func TestMemoryIssuerService(t *testing.T) {
	db, _ := testserver.NewDBForTest(t)
	t.Parallel()

	mappingStrs := map[string]string{
		"foo": "123",
	}

	mappings, err := types.NewClaimsMapping(mappingStrs)
	if err != nil {
		panic(err)
	}

	issuer := types.Issuer{
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
					ID:            issuer.ID,
					Name:          issuer.Name,
					URI:           issuer.URI,
					JWKSURI:       issuer.JWKSURI,
					ClaimMappings: mappingStrs,
				},
			},
		},
	}

	issSvc, err := newMemoryIssuerService(config)
	assert.Nil(t, err)

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
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
				CheckFn: func(t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.Nil(t, res.Err)
					compareIssuers(t, issuer, *res.Success)
				},
			},
		}

		runFn := func(ctx context.Context, input types.Issuer) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.CreateIssuer(context.Background(), input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetByURI", func(t *testing.T) {
		t.Parallel()

		testCases := []testingx.TestCase[string, *types.Issuer]{
			{
				Name:  "NotFound",
				Input: "https://evil.biz/",
				CheckFn: func(t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
			{
				Name:  "Success",
				Input: "https://example.com/",
				CheckFn: func(t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.GetIssuerByURI(context.Background(), input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetByID", func(t *testing.T) {
		t.Parallel()

		testCases := []testingx.TestCase[string, *types.Issuer]{
			{
				Name:  "NotFound",
				Input: "00000000-0000-0000-0000-000000000000",
				CheckFn: func(t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
			{
				Name:  "Success",
				Input: issuer.ID,
				CheckFn: func(t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.GetIssuerByID(context.Background(), input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			ID:            "b9ae2e16-11c0-49e4-8d9b-1d6698bba1a3",
			Name:          "Good issuer",
			URI:           "https://issuer-bba1a3.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		_, err := issSvc.CreateIssuer(context.Background(), issuer)
		assert.NoError(t, err)

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

		testCases := []testingx.TestCase[types.IssuerUpdate, *types.Issuer]{
			{
				Name:  "Full",
				Input: fullUpdate,
				CheckFn: func(t *testing.T, res testingx.TestResult[*types.Issuer]) {
					exp := issuer
					exp.Name = newName
					exp.URI = newURI
					exp.JWKSURI = newJWKSURI
					exp.ClaimMappings = newMapping

					if assert.NoError(t, res.Err) {
						compareIssuers(t, exp, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input types.IssuerUpdate) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.UpdateIssuer(context.Background(), issuer.ID, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			ID:            "ace77968-03b0-4f1b-b3f0-b214daf4ac18",
			Name:          "Good issuer",
			URI:           "https://issuer-f4ac18.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		_, err := issSvc.CreateIssuer(context.Background(), issuer)
		assert.NoError(t, err)

		testCases := []testingx.TestCase[string, any]{
			{
				Name:  "Success",
				Input: issuer.ID,
				CheckFn: func(t *testing.T, res testingx.TestResult[any]) {
					if assert.NoError(t, res.Err) {
						_, err := issSvc.GetIssuerByID(context.Background(), issuer.ID)
						assert.ErrorIs(t, types.ErrorIssuerNotFound, err)
					}
				},
			},
			{
				Name:  "NotFound",
				Input: "00000000-0000-0000-0000-000000000000",
				CheckFn: func(t *testing.T, res testingx.TestResult[any]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[any] {
			err := issSvc.DeleteIssuer(context.Background(), input)

			result := testingx.TestResult[any]{
				Success: nil,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})
}
