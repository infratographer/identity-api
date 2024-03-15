package celutils_test

import (
	"context"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-api/internal/celutils"
	"go.infratographer.com/identity-api/internal/testingx"
)

// TestClaimMappingParse checks that claim mapping expressions parse correctly.
func TestClaimMappingParse(t *testing.T) {
	t.Parallel()

	runFn := func(_ context.Context, prog string) testingx.TestResult[*cel.Ast] {
		out, Err := celutils.ParseCEL(prog)

		return testingx.TestResult[*cel.Ast]{
			Success: out,
			Err:     Err,
		}
	}

	testCases := []testingx.TestCase[string, *cel.Ast]{
		{
			Name:  "ParseError",
			Input: "'hello",
			CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[*cel.Ast]) {
				assert.Nil(t, result.Success)
				assert.NotNil(t, result.Err)
				assert.ErrorIs(t, result.Err, &celutils.ErrorCELParse{})
			},
		},
		{
			Name:  "Success",
			Input: "'hello! ' + subSHA256",
			CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[*cel.Ast]) {
				assert.Nil(t, result.Err)
				assert.NotNil(t, result.Success)
			},
		},
	}

	testingx.RunTests(context.Background(), t, testCases, runFn)
}
