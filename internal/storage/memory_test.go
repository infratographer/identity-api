package storage

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-manager-sts/internal/types"
)

func TestMemoryIssuerService(t *testing.T) {
	db, _ := testserver.NewDBForTest(t)
	t.Parallel()

	type testResult struct {
		iss *types.Issuer
		err error
	}

	type testCase struct {
		name    string
		input   string
		checkFn func(*testing.T, testResult)
	}

	issuer := types.Issuer{
		ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
		Name:          "Example",
		URI:           "https://example.com/",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: types.ClaimsMapping{},
	}

	testCases := []testCase{
		{
			name:  "NotFound",
			input: "https://evil.biz/",
			checkFn: func(t *testing.T, res testResult) {
				assert.ErrorIs(t, types.ErrorIssuerNotFound, res.err)
			},
		},
		{
			name:  "Success",
			input: "https://example.com/",
			checkFn: func(t *testing.T, res testResult) {
				assert.Nil(t, res.err)
				assert.Equal(t, &issuer, res.iss)
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

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			iss, err := issSvc.GetByURI(context.Background(), testCase.input)

			result := testResult{
				iss: iss,
				err: err,
			}

			testCase.checkFn(t, result)
		})
	}
}
