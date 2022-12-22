package celutils_test

import (
	"context"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/dmv/pkg/celutils"
)

type testResult[U any] struct {
	success U
	err     error
}

type testCase[T, U any] struct {
	name    string
	input   T
	checkFn func(*testing.T, testResult[U])
}

// TestClaimMappingParse checks that claim mapping expressions parse correctly.
func TestClaimMappingParse(t *testing.T) {
	t.Parallel()

	runFn := func(ctx context.Context, prog string) testResult[cel.Program] {
		out, err := celutils.ParseCEL(prog)

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
				assert.ErrorIs(t, result.err, &celutils.ErrorCELParse{})
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

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := runFn(context.Background(), testCase.input)
			testCase.checkFn(t, result)
		})
	}
}
