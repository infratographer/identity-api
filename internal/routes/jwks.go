package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2"

	"go.infratographer.com/identity-manager-sts/internal/fositex"
)

type jwksHandler struct {
	logger *zap.SugaredLogger
	config fositex.OAuth2Configurator
}

// Handle processes the request for the JWKS handler.
func (h *jwksHandler) Handle(ctx *gin.Context) {
	jwks := h.config.GetSigningJWKS(ctx)

	out := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{},
	}

	for _, key := range jwks.Keys {
		if public := key.Public(); public.Valid() {
			out.Keys = append(out.Keys, public)
		}
	}

	ctx.JSON(http.StatusOK, out)
}
