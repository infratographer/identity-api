package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "go.infratographer.com/dmv/pkg/api/v1"
)

func TestMemoryIssuerService(t *testing.T) {
	type testResult struct {
		iss *v1.Issuer
		err error
	}

	type testCase struct {
		name    string
		input   string
		checkFn func(*testing.T, testResult)
	}

	issuer := v1.Issuer{
		ID:      "abcd1234",
		Name:    "Example",
		URI:     "https://example.com/",
		JWKSURI: "https://example.com/.well-known/jwks.json",
	}

	testCases := []testCase{
		{
			name:  "NotFound",
			input: "https://evil.biz/",
			checkFn: func(t *testing.T, res testResult) {
				expErr := v1.ErrorIssuerNotFound{
					URI: "https://evil.biz/",
				}

				assert.ErrorIs(t, expErr, res.err)
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

	config := MemoryConfig{
		Issuers: []v1.Issuer{
			issuer,
		},
	}

	issSvc, err := newMemoryIssuerService(config)
	assert.Nil(t, err)

	for _, testCase := range testCases {
		iss, err := issSvc.GetByURI(context.Background(), testCase.input)

		result := testResult{
			iss: iss,
			err: err,
		}

		testCase.checkFn(t, result)
	}
}
