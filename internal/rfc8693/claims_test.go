package rfc8693

import (
	"context"
	"testing"

	"github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-api/internal/celutils"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/x/crdbx"
)

// TestClaimMappingEval checks that claim mapping expressions evaluate correctly.
func TestClaimMappingEval(t *testing.T) {
	t.Parallel()

	testServer, err := storage.InMemoryCRDB()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	err = testServer.Start()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	t.Cleanup(func() {
		testServer.Stop()
	})

	cm := map[string]string{
		"plusone":            "1 + claims.num",
		"infratographer:sub": "'infratographer://example.com/' + subSHA256",
	}

	config := crdbx.Config{
		URI: testServer.PGURL().String(),
	}

	seedData := storage.SeedData{
		Issuers: []storage.SeedIssuer{
			{
				TenantID:      "b8bfd705-b768-47a4-85a0-fe006f5bcfca",
				ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
				Name:          "Example",
				URI:           "https://example.com/",
				JWKSURI:       "https://example.com/.well-known/jwks.json",
				ClaimMappings: cm,
			},
		},
	}

	storageEngine, err := storage.NewEngine(config, storage.WithMigrations(), storage.WithSeedData(seedData))
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	strategy := NewClaimMappingStrategy(storageEngine)

	runFn := func(ctx context.Context, claims *jwt.JWTClaims) testingx.TestResult[jwt.JWTClaimsContainer] {
		out, err := strategy.MapClaims(ctx, claims)

		return testingx.TestResult[jwt.JWTClaimsContainer]{
			Success: out,
			Err:     err,
		}
	}

	testCases := []testingx.TestCase[*jwt.JWTClaims, jwt.JWTClaimsContainer]{
		{
			Name: "RuntimeError",
			Input: &jwt.JWTClaims{
				Subject: "foo",
				Issuer:  "https://example.com/",
				Extra:   map[string]any{},
			},
			CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[jwt.JWTClaimsContainer]) {
				assert.NotNil(t, result.Err)
				assert.ErrorIs(t, result.Err, &celutils.ErrorCELEval{})
			},
		},
		{
			Name: "MissingSub",
			Input: &jwt.JWTClaims{
				Issuer: "https://example.com/",
				Extra: map[string]any{
					"num": 1,
				},
			},
			CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[jwt.JWTClaimsContainer]) {
				assert.NotNil(t, result.Err)
				assert.ErrorIs(t, result.Err, ErrorMissingSub)
			},
		},
		{
			Name: "MissingISS",
			Input: &jwt.JWTClaims{
				Subject: "foo",
				Extra: map[string]any{
					"num": 1,
				},
			},
			CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[jwt.JWTClaimsContainer]) {
				assert.NotNil(t, result.Err)
				assert.ErrorIs(t, result.Err, ErrorMissingIss)
			},
		},
		{
			Name: "Success",
			Input: &jwt.JWTClaims{
				Subject: "foo",
				Issuer:  "https://example.com/",
				Extra: map[string]any{
					"num": 2,
				},
			},
			CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[jwt.JWTClaimsContainer]) {
				assert.Nil(t, result.Err)
				expected := &jwt.JWTClaims{
					Extra: map[string]any{
						"plusone":            int64(3),
						"infratographer:sub": "infratographer://example.com/2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
					},
				}
				assert.Equal(t, expected, result.Success)
			},
		},
	}

	testingx.RunTests(context.Background(), t, testCases, runFn)
}
