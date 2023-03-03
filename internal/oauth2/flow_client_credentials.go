// Package oauth2 provides token endpoint handlers.
package oauth2

import (
	"context"
	"fmt"
	"time"

	"github.com/ory/x/errorsx"
	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/types"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/token/jwt"
)

var _ fosite.TokenEndpointHandler = &ClientCredentialsGrantHandler{}

type clientCredentialsConfigurator interface {
	fosite.ScopeStrategyProvider
	fosite.AudienceStrategyProvider
	fosite.AccessTokenLifespanProvider
	fosite.AccessTokenIssuerProvider
	fositex.UserInfoAudienceProvider
	fositex.SigningKeyProvider
}

// ClientCredentialsGrantHandler handles the RFC6749 client credentials grant type.
type ClientCredentialsGrantHandler struct {
	*oauth2.HandleHelper
	types.UserInfoService
	storage.TransactionManager
	Config clientCredentialsConfigurator
}

// HandleTokenEndpointRequest implements https://tools.ietf.org/html/rfc6749#section-4.4.2
func (c *ClientCredentialsGrantHandler) HandleTokenEndpointRequest(ctx context.Context, request fosite.AccessRequester) error {
	client := request.GetClient()
	// The client MUST authenticate with the authorization server as described in Section 3.2.1.
	// This requirement is already fulfilled because fosite requires all token requests to be authenticated as described
	// in https://tools.ietf.org/html/rfc6749#section-3.2.1
	if client.IsPublic() {
		return errorsx.WithStack(fosite.ErrInvalidGrant.WithHint("The OAuth 2.0 Client is marked as public and is not allowed to use authorization grant 'client_credentials'."))
	}

	for _, scope := range request.GetRequestedScopes() {
		if !c.Config.GetScopeStrategy(ctx)(client.GetScopes(), scope) {
			return errorsx.WithStack(fosite.ErrInvalidScope.WithHintf("The OAuth 2.0 Client is not allowed to request scope '%s'.", scope))
		}
	}

	if err := c.Config.GetAudienceStrategy(ctx)(client.GetAudience(), request.GetRequestedAudience()); err != nil {
		return err
	}

	// First grant the /userinfo audience
	request.GrantAudience(c.Config.GetUserInfoAudience())

	// grant audiences, we checked if they were permitted above
	for _, aud := range request.GetRequestedAudience() {
		request.GrantAudience(aud)
	}

	atLifespan := fosite.GetEffectiveLifespan(client, fosite.GrantTypeClientCredentials, fosite.AccessToken, c.Config.GetAccessTokenLifespan(ctx))
	session := request.GetSession().(*oauth2.JWTSession)

	headers := jwt.Headers{}
	headers.Add("kid", c.Config.GetSigningKey(ctx).KeyID)

	session.JWTClaims = &jwt.JWTClaims{}
	session.JWTClaims.Add("client_id", request.GetClient().GetID())

	session.JWTHeader = &headers
	session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(atLifespan))

	dbCtx, err := c.BeginContext(ctx)
	if err != nil {
		return errorsx.WithStack(fosite.ErrServerError.WithHintf("could not start transaction: %v", err))
	}

	userInfo := types.UserInfo{
		Issuer:  c.Config.GetAccessTokenIssuer(ctx),
		Subject: request.GetClient().GetID(),
	}

	uiWithID, err := c.StoreUserInfo(dbCtx, userInfo)
	if err != nil {
		return errorsx.WithStack(fosite.ErrServerError.WithHintf("unable to create user info for client: %v", err))
	}

	if err := c.CommitContext(dbCtx); err != nil {
		return errorsx.WithStack(fosite.ErrServerError.WithHintf("unable to store userinfo for client: %v", err))
	}

	session.JWTClaims.Subject = fmt.Sprintf("urn:infratographer:user/%s", uiWithID.ID)

	return nil
}

// PopulateTokenEndpointResponse implements https://tools.ietf.org/html/rfc6749#section-4.4.3
func (c *ClientCredentialsGrantHandler) PopulateTokenEndpointResponse(ctx context.Context, request fosite.AccessRequester, response fosite.AccessResponder) error {
	// fosite doesn't check if this is the right handler on calls to this function.
	if !c.CanHandleTokenEndpointRequest(ctx, request) {
		return errorsx.WithStack(fosite.ErrUnknownRequest)
	}

	atLifespan := fosite.GetEffectiveLifespan(request.GetClient(), fosite.GrantTypeClientCredentials, fosite.AccessToken, c.Config.GetAccessTokenLifespan(ctx))

	return c.IssueAccessToken(ctx, atLifespan, request, response)
}

// CanSkipClientAuth determines if the client must be authenticated to use this handler.
func (c *ClientCredentialsGrantHandler) CanSkipClientAuth(ctx context.Context, requester fosite.AccessRequester) bool {
	return false
}

// CanHandleTokenEndpointRequest checks if this handler can handle the request.
func (c *ClientCredentialsGrantHandler) CanHandleTokenEndpointRequest(ctx context.Context, requester fosite.AccessRequester) bool {
	// grant_type REQUIRED.
	// Value MUST be set to "client_credentials".
	return requester.GetGrantTypes().ExactOne("client_credentials")
}

var _ fositex.Factory = NewClientCredentialsHandlerFactory

// NewClientCredentialsHandlerFactory is a fositex.Factory that
// produces a handler for the 'client_credentials' grant type.
func NewClientCredentialsHandlerFactory(config fositex.OAuth2Configurator, store any, strategy any) any {
	return &ClientCredentialsGrantHandler{
		HandleHelper: &oauth2.HandleHelper{
			AccessTokenStrategy: strategy.(oauth2.AccessTokenStrategy),
			AccessTokenStorage:  store.(oauth2.AccessTokenStorage),
			Config:              config,
		},
		UserInfoService:    store.(types.UserInfoService),
		TransactionManager: store.(storage.TransactionManager),
		Config:             config.(clientCredentialsConfigurator),
	}
}
