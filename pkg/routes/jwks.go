package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2"

	"go.infratographer.com/dmv/pkg/fositex"
)

type jwksHandler struct {
	logger *zap.SugaredLogger
	config fositex.OAuth2Configurator
}

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
