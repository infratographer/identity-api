// Package routes provides the routes for the application.
package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/metal-toolbox/auditevent/middleware/echoaudit"
	"github.com/ory/fosite"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/fositex"
)

// Option is a functional configuration option for the router
type Option func(r *Router)

// Router is the router for the application.
type Router struct {
	logger         *zap.SugaredLogger
	provider       fosite.OAuth2Provider
	config         fositex.OAuth2Configurator
	issuer         string
	auditMiddlware *echoaudit.Middleware
}

// NewRouter creates a new router
func NewRouter(opts ...Option) *Router {
	router := Router{
		logger: zap.NewNop().Sugar(),
	}

	for _, opt := range opts {
		opt(&router)
	}

	return &router
}

// WithLogger sets the logger for the router
func WithLogger(logger *zap.SugaredLogger) Option {
	return func(r *Router) {
		r.logger = logger
	}
}

// WithProvider sets the fosite provider for the router
func WithProvider(provider fosite.OAuth2Provider) Option {
	return func(r *Router) {
		r.provider = provider
	}
}

// WithOauthConfig sets the fosite oauth2 configurator for the router
func WithOauthConfig(config fositex.OAuth2Configurator) Option {
	return func(r *Router) {
		r.config = config
	}
}

// WithIssuer sets the issuer for the router
func WithIssuer(issuer string) Option {
	return func(r *Router) {
		r.issuer = issuer
	}
}

// WithAuditMiddleware sets the audit middleware for the router
func WithAuditMiddleware(mw *echoaudit.Middleware) Option {
	return func(r *Router) {
		r.auditMiddlware = mw
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

	rg.POST(
		"/token",
		tok.Handle,
		r.auditMiddlware.AuditWithType("TokenRequest"),
	)
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
