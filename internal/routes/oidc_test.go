package routes

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		headers http.Header
		request string
		path    string
		expect  string
	}{
		{
			"missing request",
			nil,
			"",
			"",
			"",
		},
		{
			"no path",
			nil,
			"http://test.local/path",
			"",
			"http://test.local/path",
		},
		{
			"with path",
			nil,
			"http://test.local/path",
			"to/test",
			"http://test.local/path/to/test",
		},
		{
			"with updir path",
			nil,
			"http://test.local/path",
			"../to/test",
			"http://test.local/to/test",
		},
		{
			"with port",
			nil,
			"http://test.local:9191/path/to/random/endpoint",
			"to/test",
			"http://test.local:9191/path/to/random/endpoint/to/test",
		},
		{
			"with path, alt hostname, trusted proxy",
			http.Header{
				"X-Forwarded-For":   []string{"1.2.3.4"},
				"X-Forwarded-Host":  []string{"test2.local"},
				"X-Forwarded-Proto": []string{"schemetest"},
			},
			"http://test.local/path",
			"to/test",
			"schemetest://test2.local/path/to/test",
		},
		{
			"with port, trusted proxy",
			http.Header{
				"X-Forwarded-For":   []string{"1.2.3.4"},
				"X-Forwarded-Proto": []string{"schemetest"},
			},
			"http://test.local:9191/path/to/random/endpoint",
			"",
			"schemetest://test.local:9191/path/to/random/endpoint",
		},
		{
			"with path, alt hostname, untrusted proxy",
			http.Header{
				"X-Forwarded-For":   []string{"1.2.3.4"},
				"X-Forwarded-Host":  []string{"test2.local"},
				"X-Forwarded-Proto": []string{"schemetest"},
			},
			"http://test.local/path",
			"to/test",
			"http://test.local/path/to/test",
		},
		{
			"with port, untrusted proxy",
			http.Header{
				"X-Forwarded-For":   []string{"1.2.3.4"},
				"X-Forwarded-Proto": []string{"schemetest"},
			},
			"http://test.local:9191/path/to/random/endpoint",
			"",
			"http://test.local:9191/path/to/random/endpoint",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.IPExtractor = echo.ExtractIPFromXFFHeader(
				echo.TrustIPRange(
					&net.IPNet{
						IP:   net.ParseIP("10.11.12.13"),
						Mask: net.CIDRMask(32, 4),
					},
				),
			)

			var request *http.Request

			if tc.request != "" {
				request = httptest.NewRequest("GET", tc.request, nil)

				// Normal http requests don't have the scheme included
				// https://stackoverflow.com/questions/40826664/get-scheme-of-the-current-request-url
				request.URL.Scheme = ""

				request.Header = tc.headers

				request.RemoteAddr = "1.2.3.4:1234"

				if strings.HasSuffix(tc.name, "trusted proxy") && !strings.HasSuffix(tc.name, "untrusted proxy") {
					request.RemoteAddr = "10.11.12.13:4567"
				}
			}

			eCtx := e.NewContext(request, httptest.NewRecorder())

			result := buildURL(eCtx, tc.path)

			if tc.expect == "" {
				require.Nil(t, result, "expected result to be nil")
			} else {
				assert.Equal(t, tc.expect, result.String(), "unexpected returned url")
			}
		})
	}
}
