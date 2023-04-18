package routes

import (
	"net"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type oidcHandler struct {
	logger *zap.SugaredLogger
	issuer string
}

type providerJSON struct {
	Issuer      string `json:"issuer"`
	AuthURL     string `json:"authorization_endpoint,omitempty"`
	TokenURL    string `json:"token_endpoint"`
	JWKSURL     string `json:"jwks_uri"`
	UserInfoURL string `json:"userinfo_endpoint"`
}

// Handle processes the request for the OIDC handler.
func (h *oidcHandler) Handle(ctx echo.Context) error {
	out := providerJSON{
		Issuer:      h.issuer,
		TokenURL:    buildURL(ctx, "../../token").String(),
		JWKSURL:     buildURL(ctx, "../../jwks.json").String(),
		UserInfoURL: buildURL(ctx, "../../userinfo").String(),
	}

	return ctx.JSON(http.StatusOK, out)
}

// buildURL returns a new *url.URL for the current page being requested, overwriting the values with the ones provided.
//
//nolint:cyclop // necessary complexity.
func buildURL(c echo.Context, path string) *url.URL {
	if c == nil || c.Request() == nil || c.Request().URL == nil {
		return nil
	}

	request := c.Request()

	outURL := &url.URL{
		Path: request.URL.JoinPath(path).Path,
	}

	remoteAddr, _, _ := net.SplitHostPort(request.RemoteAddr) //nolint:errcheck // we'll just use the empty string if an error occurs.

	// echo doesn't expose an easy way to check if the request came from a trusted proxy.
	// However the RealIP will return the source ip instead of the remote ip if coming from a trusted proxy.
	// So we can compare the two, if they're the same, then we're either not behind a proxy or not behind a trusted proxy.
	if c.RealIP() != remoteAddr {
		if scheme := request.Header.Get("X-Forwarded-Proto"); scheme != "" {
			outURL.Scheme = scheme
		}

		if host := request.Header.Get("X-Forwarded-Host"); host != "" {
			outURL.Host = host
		}
	}

	if outURL.Scheme == "" {
		// Request.URL.Scheme is usually empty, however if it isn't we'll use that scheme.
		// If empty, we'll check if TLS was used, and if so, set the scheme as https.
		switch {
		case request.URL.Scheme != "":
			outURL.Scheme = request.URL.Scheme
		case request.TLS != nil:
			outURL.Scheme = "https"
		default:
			outURL.Scheme = "http"
		}
	}

	if outURL.Host == "" {
		if host := request.Host; host != "" {
			outURL.Host = host
		}
	}

	return outURL
}
