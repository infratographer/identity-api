// Package userinfo contains the enpdoints for translating
// STS tokens to original IdP user info.
package userinfo

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.hollow.sh/toolbox/ginauth"
	"go.hollow.sh/toolbox/ginjwt"
	"gopkg.in/square/go-jose.v2"

	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/rfc8693"
	"go.infratographer.com/identity-api/internal/types"
)

// Handler provides the endpoint for /userinfo
type Handler struct {
	store types.UserInfoService
	mw    *ginauth.MultiTokenMiddleware
}

// NewHandler creates a UserInfo handler with the storage engine
func NewHandler(userInfoSvc types.UserInfoService, cfg fositex.OAuth2Configurator) (*Handler, error) {
	ctx := context.Background()

	jwks := cfg.GetSigningJWKS(ctx)

	set := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{},
	}

	for _, key := range jwks.Keys {
		if public := key.Public(); public.Valid() {
			set.Keys = append(set.Keys, public)
		}
	}

	issuer := cfg.GetAccessTokenIssuer(ctx)

	audience, err := url.JoinPath(issuer, "userinfo")
	if err != nil {
		return nil, err
	}

	userInfoAuthCfg := ginjwt.AuthConfig{
		Enabled:                true,
		Audience:               audience,
		Issuer:                 issuer,
		JWKS:                   set,
		RoleValidationStrategy: "all",
	}

	mw, err := ginjwt.NewMultiTokenMiddlewareFromConfigs(userInfoAuthCfg)
	if err != nil {
		return nil, err
	}

	return &Handler{
		store: userInfoSvc,
		mw:    mw,
	}, nil
}

// Handle expects an authenticated request using a STS token and returns
// the stored userinfo if it exists.
func (h *Handler) handle(ctx *gin.Context) {
	fullSubject := ginjwt.GetSubject(ctx)

	prefix := rfc8693.SubjectPrefix + "/"
	parts := strings.SplitAfter(fullSubject, prefix)

	info, err := h.store.LookupUserInfoByID(ctx.Request.Context(), parts[1])
	if err != nil {
		out := map[string]any{
			"errors": []string{err.Error()},
		}
		ctx.AbortWithStatusJSON(http.StatusBadRequest, out)

		return
	}

	ctx.JSON(http.StatusOK, info)
}

// Routes registers the userinfo handler in a gin.RouterGroup
func (h *Handler) Routes(rg *gin.RouterGroup) {
	authMw := h.mw.AuthRequired([]string{})
	rg.GET("userinfo", authMw, h.handle)
}
