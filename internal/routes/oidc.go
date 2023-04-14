package routes

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
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
func (h *oidcHandler) Handle(ctx *gin.Context) {
	out := providerJSON{
		Issuer:      h.issuer,
		TokenURL:    buildURL(ctx, "../../token").String(),
		JWKSURL:     buildURL(ctx, "../../jwks.json").String(),
		UserInfoURL: buildURL(ctx, "../../userinfo").String(),
	}

	ctx.JSON(http.StatusOK, out)
}

// buildURL returns a new *url.URL for the current page being requested, overwriting the values with the ones provided.
//
//nolint:cyclop // necessary complexity.
func buildURL(c *gin.Context, path string) *url.URL {
	outURL := new(url.URL)

	if c != nil && c.Request != nil && c.Request.URL != nil {
		outURL.Path = c.Request.URL.JoinPath(path).Path

		// gin doesn't expose an easy way to check if the request came from a trusted proxy.
		// However the ClientIP will return the source ip instead of the remote ip if coming from a trusted proxy.
		// So we can compare the two, if they're the same, then we're either not behind a proxy or not behind a trusted proxy.
		if c.ClientIP() != c.RemoteIP() {
			if scheme := c.Request.Header.Get("X-Forwarded-Proto"); scheme != "" {
				outURL.Scheme = scheme
			}

			if host := c.Request.Header.Get("X-Forwarded-Host"); host != "" {
				outURL.Host = host
			}
		}

		if outURL.Scheme == "" {
			// Request.URL.Scheme is usually empty, however if it isn't we'll use that scheme.
			// If empty, we'll check if TLS was used, and if so, set the scheme as https.
			switch {
			case c.Request.URL.Scheme != "":
				outURL.Scheme = c.Request.URL.Scheme
			case c.Request.TLS != nil:
				outURL.Scheme = "https"
			default:
				outURL.Scheme = "http"
			}
		}

		if outURL.Host == "" {
			if host := c.Request.Host; host != "" {
				outURL.Host = host
			}
		}
	}

	return outURL
}
