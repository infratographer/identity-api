// Package routes provides the routes for the application.
package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ory/fosite"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/fositex"
)

// Router is the router for the application.
type Router struct {
	logger   *zap.SugaredLogger
	provider fosite.OAuth2Provider
	config   fositex.OAuth2Configurator
	issuer   string
}

// NewRouter creates a new Router.
func NewRouter(logger *zap.SugaredLogger, config fositex.OAuth2Configurator, provider fosite.OAuth2Provider, issuer string) *Router {
	return &Router{
		logger:   logger,
		provider: provider,
		config:   config,
		issuer:   issuer,
	}
}

// Routes registers the routes for the application.
func (r *Router) Routes(rg *echo.Group) {
	tok := &tokenHandler{
		logger:   r.logger,
		provider: r.provider,
	}
	jwks := &jwksHandler{
		logger: r.logger,
		config: r.config,
	}
	oidc := &oidcHandler{
		logger: r.logger,
		issuer: r.issuer,
	}

	rg.POST("/token", tok.Handle)
	rg.GET("/jwks.json", jwks.Handle)
	rg.GET("/.well-known/openid-configuration", oidc.Handle)
}

// SkipNoAuthRoutes returns true if the requesting path should not have auth validated for it.
func SkipNoAuthRoutes(c echo.Context) bool {
	switch c.Request().URL.Path {
	case "/token", "/jwks.json", "/.well-known/openid-configuration":
		return true
	default:
		return false
	}
}
