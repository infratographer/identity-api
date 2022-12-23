package rfc8693

import (
	"context"
	"testing"

	"github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/dmv/pkg/storage"
)

type testFunc[T, U any] func(context.Context, T) testResult[U]

type testResult[U any] struct {
	success U
	err     error
}

type testCase[T, U any] struct {
	name    string
	input   T
	checkFn func(*testing.T, testResult[U])
}

func runTests[T, U any](ctx context.Context, t *testing.T, cases []testCase[T, U], testFn testFunc[T, U]) {
	for _, testCase := range cases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := testFn(ctx, testCase.input)
			testCase.checkFn(t, result)
		})
	}
}

// TestClaimMappingEval checks that claim mapping expressions evaluate correctly.
func TestClaimMappingEval(t *testing.T) {
	t.Parallel()

	cm := map[string]string{
		"plusone":            "1 + claims.num",
		"infratographer:sub": "'infratographer://example.com/' + subSHA256",
	}

	cfg := storage.Config{
		Type: storage.EngineTypeMemory,
		SeedData: storage.SeedData{
			Issuers: []storage.SeedIssuer{
				{
					ID:            "abcd1234",
					Name:          "Example",
					URI:           "https://example.com/",
					JWKSURI:       "https://example.com/.well-known/jwks.json",
					ClaimMappings: cm,
				},
			},
		},
	}

	storageEngine, err := storage.NewEngine(cfg)
	assert.NoError(t, err, "failed to create storage engine")

	strategy := NewClaimMappingStrategy(storageEngine)

	runFn := func(ctx context.Context, claims *jwt.JWTClaims) testResult[jwt.JWTClaimsContainer] {
		out, err := strategy.MapClaims(ctx, claims)

		return testResult[jwt.JWTClaimsContainer]{
			success: out,
			err:     err,
		}
	}

	testCases := []testCase[*jwt.JWTClaims, jwt.JWTClaimsContainer]{
		{
			name: "RuntimeError",
			input: &jwt.JWTClaims{
				Subject: "foo",
				Issuer:  "https://example.com/",
				Extra:   map[string]any{},
			},
			checkFn: func(t *testing.T, result testResult[jwt.JWTClaimsContainer]) {
				assert.NotNil(t, result.err)
				assert.ErrorIs(t, result.err, &ErrorCELEval{})
			},
		},
		{
			name: "MissingSub",
			input: &jwt.JWTClaims{
				Issuer: "https://example.com/",
				Extra: map[string]any{
					"num": 1,
				},
			},
			checkFn: func(t *testing.T, result testResult[jwt.JWTClaimsContainer]) {
				assert.NotNil(t, result.err)
				assert.ErrorIs(t, result.err, ErrorMissingSub)
			},
		},
		{
			name: "MissingISS",
			input: &jwt.JWTClaims{
				Subject: "foo",
				Extra: map[string]any{
					"num": 1,
				},
			},
			checkFn: func(t *testing.T, result testResult[jwt.JWTClaimsContainer]) {
				assert.NotNil(t, result.err)
				assert.ErrorIs(t, result.err, ErrorMissingIss)
			},
		},
		{
			name: "Success",
			input: &jwt.JWTClaims{
				Subject: "foo",
				Issuer:  "https://example.com/",
				Extra: map[string]any{
					"num": 2,
				},
			},
			checkFn: func(t *testing.T, result testResult[jwt.JWTClaimsContainer]) {
				assert.Nil(t, result.err)
				expected := &jwt.JWTClaims{
					Extra: map[string]any{
						"plusone":            int64(3),
						"infratographer:sub": "infratographer://example.com/2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
					},
				}
				assert.Equal(t, expected, result.success)
			},
		},
	}

	runTests(context.Background(), t, testCases, runFn)
}
