package routes

import (
	"net/http"

	jose "github.com/go-jose/go-jose/v3"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/fositex"
)

type jwksHandler struct {
	logger *zap.SugaredLogger
	config fositex.OAuth2Configurator
}

// Handle processes the request for the JWKS handler.
func (h *jwksHandler) Handle(ctx echo.Context) error {
	jwks := h.config.GetSigningJWKS(ctx.Request().Context())

	out := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{},
	}

	for _, key := range jwks.Keys {
		if public := key.Public(); public.Valid() {
			out.Keys = append(out.Keys, public)
		}
	}

	return ctx.JSON(http.StatusOK, out)
}
