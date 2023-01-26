package storage

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-manager-sts/internal/testingx"
	"go.infratographer.com/identity-manager-sts/internal/types"
)

func TestMemoryIssuerService(t *testing.T) {
	db, _ := testserver.NewDBForTest(t)
	t.Parallel()

	issuer := types.Issuer{
		ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
		Name:          "Example",
		URI:           "https://example.com/",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: types.ClaimsMapping{},
	}

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
				assert.Nil(t, res.Err)
				assert.Equal(t, &issuer, res.Success)
			},
		},
	}

	config := Config{
		db: db,
		SeedData: SeedData{
			Issuers: []SeedIssuer{
				{
					ID:      issuer.ID,
					Name:    issuer.Name,
					URI:     issuer.URI,
					JWKSURI: issuer.JWKSURI,
				},
			},
		},
	}

	issSvc, err := newMemoryIssuerService(config)
	assert.Nil(t, err)

	runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
		iss, err := issSvc.GetIssuerByURI(context.Background(), input)

		result := testingx.TestResult[*types.Issuer]{
			Success: iss,
			Err:     err,
		}

		return result
	}

	testingx.RunTests(context.Background(), t, testCases, runFn)
}
