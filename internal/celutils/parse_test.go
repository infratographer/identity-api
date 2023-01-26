package celutils_test

import (
	"context"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-manager-sts/internal/celutils"
	"go.infratographer.com/identity-manager-sts/internal/testingx"
)

// TestClaimMappingParse checks that claim mapping expressions parse correctly.
func TestClaimMappingParse(t *testing.T) {
	t.Parallel()

	runFn := func(ctx context.Context, prog string) testingx.TestResult[*cel.Ast] {
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
			CheckFn: func(t *testing.T, result testingx.TestResult[*cel.Ast]) {
				assert.Nil(t, result.Success)
				assert.NotNil(t, result.Err)
				assert.ErrorIs(t, result.Err, &celutils.ErrorCELParse{})
			},
		},
		{
			Name:  "Success",
			Input: "'hello! ' + subSHA256",
			CheckFn: func(t *testing.T, result testingx.TestResult[*cel.Ast]) {
				assert.Nil(t, result.Err)
				assert.NotNil(t, result.Success)
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()

			result := runFn(context.Background(), testCase.Input)
			testCase.CheckFn(t, result)
		})
	}
}
