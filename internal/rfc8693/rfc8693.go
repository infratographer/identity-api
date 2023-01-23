// Package rfc8693 implements the token exchange grant type per RFC 8693.
package rfc8693

import (
	"context"
	"errors"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/token/jwt"
	"github.com/ory/x/errorsx"

	"go.infratographer.com/identity-manager-sts/internal/fositex"
)

const (
	// GrantTypeTokenExchange is the grant type for token exchange per RFC 8693.
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	// TokenTypeJWT is the token type for JWT per RFC 8693.
	TokenTypeJWT = "urn:ietf:params:oauth:token-type:jwt"
	// ParamSubjectToken is the OAuth 2.0 request parameter for the subject token.
	ParamSubjectToken = "subject_token"
	// ParamSubjectTokenType is the OAuth 2.0 request parameter for the subject token type.
	ParamSubjectTokenType = "subject_token_type"
	// ParamActorToken is the OAuth 2.0 request parameter for the actor token.
	ParamActorToken = "actor_token"
	// ParamActorTokenType is the OAuth 2.0 request parameter for the actor token type.
	ParamActorTokenType = "actor_token_type"
	// ClaimClientID is the claim for the client ID.
	ClaimClientID = "client_id"
)

var (
	// ErrJWKSURIStrategyNotDefined is returned when the issuer JWKS URI strategy is not defined.
	ErrJWKSURIStrategyNotDefined = errors.New("no issuer JWKS URI strategy defined")
)

func findMatchingKey(ctx context.Context, config fositex.OAuth2Configurator, token *jwt.Token) (interface{}, error) {
	var claims jwt.JWTClaims

	claims.FromMapClaims(token.Claims)

	issuer := claims.Issuer
	if len(issuer) == 0 {
		return nil, &jwt.ValidationError{
			Errors: jwt.ValidationErrorIssuer,
		}
	}

	jwksURIStrategy := config.GetIssuerJWKSURIStrategy(ctx)
	if jwksURIStrategy == nil {
		return nil, &jwt.ValidationError{
			Errors: jwt.ValidationErrorUnverifiable,
			Inner:  ErrJWKSURIStrategyNotDefined,
		}
	}

	jwksURI, err := jwksURIStrategy.GetIssuerJWKSURI(ctx, issuer)
	if err != nil {
		return nil, &jwt.ValidationError{
			Errors: jwt.ValidationErrorIssuer,
			Inner:  err,
		}
	}

	jwks, err := config.GetJWKSFetcherStrategy(ctx).Resolve(ctx, jwksURI, false)
	if err != nil {
		return nil, &jwt.ValidationError{
			Errors: jwt.ValidationErrorUnverifiable,
			Inner:  err,
		}
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, &jwt.ValidationError{
			Errors: jwt.ValidationErrorMalformed,
		}
	}

	keys := jwks.Key(kid)

	for _, key := range keys {
		if key.Use == "sig" {
			return key, nil
		}
	}

	err = &jwt.ValidationError{
		Errors: jwt.ValidationErrorSignatureInvalid,
	}

	return nil, err
}

// TokenExchangeHandler contains the logic for the token exchange grant type.
// it implements the fosite.TokenEndpointHandler interface.
type TokenExchangeHandler struct {
	accessTokenStrategy oauth2.AccessTokenStrategy
	accessTokenStorage  oauth2.AccessTokenStorage
	config              fositex.OAuth2Configurator
}

// implement the fosite.TokenEndpointHandler interface
var _ fosite.TokenEndpointHandler = new(TokenExchangeHandler)

// NewTokenExchangeHandler creates a new TokenExchangeHandler.
func NewTokenExchangeHandler(config fositex.OAuth2Configurator, strategy oauth2.AccessTokenStrategy, storage oauth2.AccessTokenStorage) *TokenExchangeHandler {
	return &TokenExchangeHandler{
		accessTokenStrategy: strategy,
		accessTokenStorage:  storage,
		config:              config,
	}
}

func (s *TokenExchangeHandler) validateJWT(ctx context.Context, token string, strategy fosite.JWKSFetcherStrategy) (*jwt.Token, error) {
	// Side effectful key finding isn't great but neither is parsing the JWT twice
	keyfunc := func(token *jwt.Token) (interface{}, error) {
		return findMatchingKey(ctx, s.config, token)
	}

	parsed, err := jwt.Parse(token, keyfunc)

	if err == nil {
		return parsed, nil
	}

	validationErr, ok := err.(*jwt.ValidationError)
	if !ok {
		return nil, errorsx.WithStack(fosite.ErrServerError.WithDebugf("Unknown error: %s", err))
	}

	switch validationErr.Errors {
	case jwt.ValidationErrorUnverifiable:
		return nil, errorsx.WithStack(fosite.ErrServerError.WithHintf("Server error: %s", err))
	default:
		return nil, errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("Invalid subject token: %s", err))
	}
}

func (s *TokenExchangeHandler) getSubjectClaims(ctx context.Context, token string) (*jwt.JWTClaims, error) {
	validated, err := s.validateJWT(ctx, token, s.config.GetJWKSFetcherStrategy(ctx))

	if err != nil {
		return nil, err
	}

	var claims jwt.JWTClaims

	claims.FromMapClaims(validated.Claims)

	return &claims, nil
}

func (s *TokenExchangeHandler) getMappedSubjectClaims(ctx context.Context, claims *jwt.JWTClaims) (jwt.JWTClaimsContainer, error) {
	mappingStrategy := s.config.GetClaimMappingStrategy(ctx)

	mappedClaims, err := mappingStrategy.MapClaims(ctx, claims)
	if err != nil {
		return nil, err
	}

	return mappedClaims, nil
}

// HandleTokenEndpointRequest handles a RFC 8693 token request and provides a response that can be used to
// generate a token. Currently only supports JWT subject tokens and impersonation semantics.
func (s *TokenExchangeHandler) HandleTokenEndpointRequest(ctx context.Context, requester fosite.AccessRequester) error {
	form := requester.GetRequestForm()

	subjectToken := form.Get(ParamSubjectToken)
	if len(subjectToken) == 0 {
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("Missing required parameter '%s'.", ParamSubjectToken))
	}

	subjectTokenType := form.Get(ParamSubjectTokenType)
	if len(subjectTokenType) == 0 {
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("Missing required parameter '%s'.", ParamSubjectTokenType))
	}

	actorToken := form.Get(ParamActorToken)
	if len(actorToken) > 0 {
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("Delegation is not supported by this handler."))
	}

	switch subjectTokenType {
	case TokenTypeJWT:
		break
	default:
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("Unsupported subject token type '%s'.", subjectTokenType))
	}

	claims, err := s.getSubjectClaims(ctx, subjectToken)
	if err != nil {
		return err
	}

	mappedClaims, err := s.getMappedSubjectClaims(ctx, claims)
	if err != nil {
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("error mapping claims: %s", err))
	}

	var newClaims jwt.JWTClaims
	newClaims.Subject = claims.Subject
	newClaims.Issuer = s.config.GetAccessTokenIssuer(ctx)

	for k, v := range mappedClaims.ToMapClaims() {
		newClaims.Add(k, v)
	}

	expiry := time.Now().Add(s.config.GetAccessTokenLifespan(ctx))
	expiryMap := map[fosite.TokenType]time.Time{
		fosite.AccessToken: expiry,
	}

	clientID := requester.GetClient().GetID()
	newClaims.Add(ClaimClientID, clientID)

	kid := s.config.GetSigningKey(ctx).KeyID

	headers := jwt.Headers{}
	headers.Add("kid", kid)

	session := oauth2.JWTSession{
		JWTHeader: &headers,
		JWTClaims: &newClaims,
		ExpiresAt: expiryMap,
		Subject:   claims.Subject,
	}

	requester.SetSession(&session)

	return nil
}

// PopulateTokenEndpointResponse populates the response with a token.
func (s *TokenExchangeHandler) PopulateTokenEndpointResponse(ctx context.Context, requester fosite.AccessRequester, responder fosite.AccessResponder) error {
	token, _, err := s.accessTokenStrategy.GenerateAccessToken(ctx, requester)
	if err != nil {
		return err
	}

	responder.SetAccessToken(token)
	responder.SetTokenType(TokenTypeJWT)
	responder.SetExpiresIn(s.config.GetAccessTokenLifespan(ctx))

	return nil
}

// CanSkipClientAuth is currently not supported by this handler.
// It returns false.
func (s *TokenExchangeHandler) CanSkipClientAuth(ctx context.Context, requester fosite.AccessRequester) bool {
	return false
}

// CanHandleTokenEndpointRequest returns true if the grant type is token exchange.
func (s *TokenExchangeHandler) CanHandleTokenEndpointRequest(ctx context.Context, requester fosite.AccessRequester) bool {
	return requester.GetGrantTypes().ExactOne(GrantTypeTokenExchange)
}
