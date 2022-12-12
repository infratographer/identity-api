package routes

import (
	"github.com/gin-gonic/gin"
	"go.infratographer.com/dmv/pkg/fositex"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2"
	"net/http"
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
