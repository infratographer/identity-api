package rfc8693

import (
	"context"
	"fmt"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/token/jwt"
	"github.com/ory/x/errorsx"

	"go.infratographer.com/dmv/pkg/fositex"
)

const (
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	TokenTypeJWT           = "urn:ietf:params:oauth:token-type:jwt"
	ParamSubjectToken      = "subject_token"
	ParamSubjectTokenType  = "subject_token_type"
	ParamActorToken        = "actor_token"
	ParamActorTokenType    = "actor_token_type"
	ClaimClientID          = "client_id"
)

func findMatchingKey(ctx context.Context, config fositex.OAuth2Configurator, token *jwt.Token) (interface{}, error) {
	var claims jwt.JWTClaims
	claims.FromMapClaims(token.Claims)

	issuer := claims.Issuer
	if len(issuer) == 0 {
		err := &jwt.ValidationError{
			Errors: jwt.ValidationErrorIssuer,
		}
		return nil, err
	}

	jwksURIStrategy := config.GetIssuerJWKSURIStrategy(ctx)
	if jwksURIStrategy == nil {
		err := &jwt.ValidationError{
			Errors: jwt.ValidationErrorUnverifiable,
			Inner:  fmt.Errorf("No issuer JWKS URI strategy defined"),
		}
		return nil, err
	}

	jwksURI, err := jwksURIStrategy.GetIssuerJWKSURI(ctx, issuer)
	if err != nil {
		wrappedErr := &jwt.ValidationError{
			Errors: jwt.ValidationErrorIssuer,
			Inner:  err,
		}
		return nil, wrappedErr
	}

	jwks, err := config.GetJWKSFetcherStrategy(ctx).Resolve(ctx, jwksURI, false)
	if err != nil {
		wrappedErr := &jwt.ValidationError{
			Errors: jwt.ValidationErrorUnverifiable,
			Inner:  err,
		}
		return nil, wrappedErr
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		err := &jwt.ValidationError{
			Errors: jwt.ValidationErrorMalformed,
		}
		return nil, err
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

type TokenExchangeHandler struct {
	accessTokenStrategy oauth2.AccessTokenStrategy
	accessTokenStorage  oauth2.AccessTokenStorage
	config              fositex.OAuth2Configurator
}

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

func (s *TokenExchangeHandler) getSubjectClaims(ctx context.Context, token string) (jwt.JWTClaims, error) {
	validated, err := s.validateJWT(ctx, token, s.config.GetJWKSFetcherStrategy(ctx))

	if err != nil {
		return jwt.JWTClaims{}, err
	}

	var claims jwt.JWTClaims
	claims.FromMapClaims(validated.Claims)

	return claims, nil
}

// HandleTokenExchangeRequest handles a RFC 8693 token request and provides a response that can be used to
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

	expiry := time.Now().Add(s.config.GetAccessTokenLifespan(ctx))
	expiryMap := map[fosite.TokenType]time.Time{
		fosite.AccessToken: expiry,
	}

	newClaims := jwt.JWTClaims{
		Subject: claims.Subject,
		Issuer:  s.config.GetAccessTokenIssuer(ctx),
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

func (s *TokenExchangeHandler) CanSkipClientAuth(ctx context.Context, requester fosite.AccessRequester) bool {
	return false
}

func (s *TokenExchangeHandler) CanHandleTokenEndpointRequest(ctx context.Context, requester fosite.AccessRequester) bool {
	return requester.GetGrantTypes().ExactOne(GrantTypeTokenExchange)
}
