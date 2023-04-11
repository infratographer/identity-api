// Package oauth2 provides token endpoint handlers.
package oauth2

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ory/x/errorsx"
	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/types"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/token/jwt"
	"go.infratographer.com/x/urnx"
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

func buildClientURN(c fosite.Client) string {
	clientUUID := uuid.MustParse(c.GetID())

	urn, err := urnx.Build(types.URNNamespace, types.URNResourceTypeClient, clientUUID)
	if err != nil {
		// If for some reason we aren't building valid URNs, panic
		panic(err)
	}

	return urn.String()
}

// ClientCredentialsGrantHandler handles the RFC6749 client credentials grant type.
type ClientCredentialsGrantHandler struct {
	*oauth2.HandleHelper
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

	requestedResources := request.GetRequestForm()["resource"]

	resources := make([]string, 0)
	for _, r := range requestedResources {
		resources = append(resources, string(r))
	}

	if err := c.Config.GetAudienceStrategy(ctx)(client.GetAudience(), fosite.Arguments(resources)); err != nil {
		return err
	}

	for _, aud := range resources {
		request.GrantAudience(aud)
	}

	atLifespan := fosite.GetEffectiveLifespan(client, fosite.GrantTypeClientCredentials, fosite.AccessToken, c.Config.GetAccessTokenLifespan(ctx))
	session := request.GetSession().(*oauth2.JWTSession)

	headers := jwt.Headers{}
	headers.Add("kid", c.Config.GetSigningKey(ctx).KeyID)

	clientID := client.GetID()

	session.JWTClaims = &jwt.JWTClaims{}
	session.JWTClaims.Add("client_id", clientID)

	session.JWTHeader = &headers
	session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(atLifespan))

	session.JWTClaims.Subject = buildClientURN(client)

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
func (c *ClientCredentialsGrantHandler) CanSkipClientAuth(_ context.Context, _ fosite.AccessRequester) bool {
	return false
}

// CanHandleTokenEndpointRequest checks if this handler can handle the request.
func (c *ClientCredentialsGrantHandler) CanHandleTokenEndpointRequest(_ context.Context, requester fosite.AccessRequester) bool {
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
		TransactionManager: store.(storage.TransactionManager),
		Config:             config.(clientCredentialsConfigurator),
	}
}
