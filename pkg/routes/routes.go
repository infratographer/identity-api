package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"go.infratographer.com/dmv/pkg/fositex"
	"go.uber.org/zap"
)

type Router struct {
	logger   *zap.SugaredLogger
	provider fosite.OAuth2Provider
	config   fositex.OAuth2Configurator
}

func NewRouter(logger *zap.SugaredLogger, config fositex.OAuth2Configurator, provider fosite.OAuth2Provider) *Router {
	return &Router{
		logger:   logger,
		provider: provider,
		config:   config,
	}
}

func (r *Router) Routes(rg *gin.RouterGroup) {
	tok := &tokenHandler{
		logger:   r.logger,
		provider: r.provider,
	}
	jwks := &jwksHandler{
		logger: r.logger,
		config: r.config,
	}

	rg.POST("/token", tok.Handle)
	rg.GET("/jwks.json", jwks.Handle)
}
