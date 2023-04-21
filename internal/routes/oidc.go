package routes

import (
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
	issuer, err := url.Parse(h.issuer)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "bad issuer").SetInternal(err)
	}

	out := providerJSON{
		Issuer:      h.issuer,
		TokenURL:    issuer.JoinPath("/token").String(),
		JWKSURL:     issuer.JoinPath("/jwks.json").String(),
		UserInfoURL: issuer.JoinPath("/userinfo").String(),
	}

	return ctx.JSON(http.StatusOK, out)
}
