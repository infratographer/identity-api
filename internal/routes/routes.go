// Package routes provides the routes for the application.
package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/metal-toolbox/auditevent/ginaudit"
	"github.com/ory/fosite"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/fositex"
)

// Router is the router for the application.
type Router struct {
	logger         *zap.SugaredLogger
	provider       fosite.OAuth2Provider
	config         fositex.OAuth2Configurator
	auditMiddlware *ginaudit.Middleware
}

// NewRouter creates a new Router.
func NewRouter(logger *zap.SugaredLogger, config fositex.OAuth2Configurator, provider fosite.OAuth2Provider, admw *ginaudit.Middleware) *Router {
	return &Router{
		logger:         logger,
		provider:       provider,
		config:         config,
		auditMiddlware: admw,
	}
}

// Routes registers the routes for the application.
func (r *Router) Routes(rg *gin.RouterGroup) {
	tok := &tokenHandler{
		logger:   r.logger,
		provider: r.provider,
	}
	jwks := &jwksHandler{
		logger: r.logger,
		config: r.config,
	}

	rg.POST(
		"/token",
		r.auditMiddlware.AuditWithType("TokenExchange"),
		tok.Handle,
	)

	rg.GET("/jwks.json", jwks.Handle)
}
