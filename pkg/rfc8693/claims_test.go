package rfc8693

import (
	"context"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/assert"
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
		t.Run(testCase.name, func(t *testing.T) {
			result := testFn(ctx, testCase.input)
			testCase.checkFn(t, result)
		})
	}
}

// TestClaimMappingParse checks that claim mapping expressions parse correctly.
func TestClaimMappingParse(t *testing.T) {
	runFn := func(ctx context.Context, prog string) testResult[cel.Program] {
		out, err := parseCEL(prog)

		return testResult[cel.Program]{
			success: out,
			err:     err,
		}
	}

	testCases := []testCase[string, cel.Program]{
		{
			name:  "ParseError",
			input: "'hello",
			checkFn: func(t *testing.T, result testResult[cel.Program]) {
				assert.Nil(t, result.success)
				assert.NotNil(t, result.err)
				assert.ErrorIs(t, result.err, &ErrorCELParse{})
			},
		},
		{
			name:  "Success",
			input: "'hello! ' + subSHA256",
			checkFn: func(t *testing.T, result testResult[cel.Program]) {
				assert.Nil(t, result.err)
				assert.NotNil(t, result.success)
			},
		},
	}

	runTests(context.Background(), t, testCases, runFn)
}

// TestClaimMappingEval checks that claim mapping expressions evaluate correctly.
func TestClaimMappingEval(t *testing.T) {
	mappingExprs := map[string]string{
		"plusone":            "1 + claims.num",
		"infratographer:sub": "'infratographer://example.com/' + subSHA256",
	}

	strategy, err := NewClaimMappingStrategy(mappingExprs)
	assert.Nil(t, err)

	runFn := func(ctx context.Context, claims jwt.JWTClaims) testResult[jwt.JWTClaims] {
		out, err := strategy.MapClaims(claims)
		return testResult[jwt.JWTClaims]{
			success: out,
			err:     err,
		}
	}

	testCases := []testCase[jwt.JWTClaims, jwt.JWTClaims]{
		{
			name: "RuntimeError",
			input: jwt.JWTClaims{
				Subject: "foo",
				Extra:   map[string]any{},
			},
			checkFn: func(t *testing.T, result testResult[jwt.JWTClaims]) {
				assert.NotNil(t, result.err)
				assert.ErrorIs(t, result.err, &ErrorCELEval{})
			},
		},
		{
			name: "Success",
			input: jwt.JWTClaims{
				Subject: "foo",
				Extra: map[string]any{
					"num": 2,
				},
			},
			checkFn: func(t *testing.T, result testResult[jwt.JWTClaims]) {
				assert.Nil(t, result.err)
				expected := jwt.JWTClaims{
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
