package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		issuer string
		expect providerJSON
	}{
		{
			"root with slash",
			"https://test.local/",
			providerJSON{
				Issuer:      "https://test.local/",
				TokenURL:    "https://test.local/token",
				JWKSURL:     "https://test.local/jwks.json",
				UserInfoURL: "https://test.local/userinfo",
			},
		},
		{
			"root without slash",
			"https://test.local",
			providerJSON{
				Issuer:      "https://test.local",
				TokenURL:    "https://test.local/token",
				JWKSURL:     "https://test.local/jwks.json",
				UserInfoURL: "https://test.local/userinfo",
			},
		},
		{
			"subpath with slash",
			"https://test.local/some/path/",
			providerJSON{
				Issuer:      "https://test.local/some/path/",
				TokenURL:    "https://test.local/some/path/token",
				JWKSURL:     "https://test.local/some/path/jwks.json",
				UserInfoURL: "https://test.local/some/path/userinfo",
			},
		},
		{
			"subpath without slash",
			"https://test.local/some/path",
			providerJSON{
				Issuer:      "https://test.local/some/path",
				TokenURL:    "https://test.local/some/path/token",
				JWKSURL:     "https://test.local/some/path/jwks.json",
				UserInfoURL: "https://test.local/some/path/userinfo",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := oidcHandler{
				issuer: tc.issuer,
			}

			e := echo.New()
			e.GET("/oidc", handler.Handle)

			w := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/oidc", nil)

			require.NoError(t, err, "unexpected error for new request")

			e.ServeHTTP(w, req)

			var got providerJSON

			err = json.NewDecoder(w.Body).Decode(&got)

			require.NoError(t, err, "unexpected error decoding response")

			assert.Equal(t, tc.expect, got, "unexpected response from request")
		})
	}
}
