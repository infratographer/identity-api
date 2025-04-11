// Package rfc8693 implements the token exchange grant type per RFC 8693.
package rfc8693

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/token/jwt"
	"github.com/ory/x/errorsx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.infratographer.com/x/gidx"

	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/types"
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

	responseIssuedTokenType = "issued_token_type"

	instrumentationName = "go.infratographer.com/identity-api/internal/rfc6893"
)

var (
	// ErrJWKSURIProviderNotDefined is returned when the issuer JWKS URI provider is not defined.
	ErrJWKSURIProviderNotDefined = errors.New("no issuer JWKS URI provider defined")
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

	jwksURIProvider := config.GetIssuerJWKSURIProvider(ctx)
	if jwksURIProvider == nil {
		return nil, &jwt.ValidationError{
			Errors: jwt.ValidationErrorUnverifiable,
			Inner:  ErrJWKSURIProviderNotDefined,
		}
	}

	jwksURI, err := jwksURIProvider.GetIssuerJWKSURI(ctx, issuer)
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
	tracer              trace.Tracer
	accessTokenStrategy oauth2.AccessTokenStrategy
	accessTokenStorage  oauth2.AccessTokenStorage
	config              fositex.OAuth2Configurator
}

// implement the fosite.TokenEndpointHandler interface
var _ fosite.TokenEndpointHandler = new(TokenExchangeHandler)

// NewTokenExchangeHandler works as a fositex.Factory to register this handler.
var _ fositex.Factory = NewTokenExchangeHandler

// NewTokenExchangeHandler creates a new TokenExchangeHandler,
func NewTokenExchangeHandler(config fositex.OAuth2Configurator, storage any, strategy any) any {
	tracer := otel.Tracer(instrumentationName)

	return &TokenExchangeHandler{
		tracer:              tracer,
		accessTokenStrategy: strategy.(oauth2.AccessTokenStrategy),
		accessTokenStorage:  storage.(oauth2.AccessTokenStorage),
		config:              config,
	}
}

func (s *TokenExchangeHandler) validateJWT(ctx context.Context, token string) (*jwt.Token, error) {
	// Side effectful key finding isn't great but neither is parsing the JWT twice
	keyfunc := func(token *jwt.Token) (interface{}, error) {
		return findMatchingKey(ctx, s.config, token)
	}

	parsed, err := jwt.Parse(token, keyfunc)

	if err == nil {
		return parsed, nil
	}

	var claims jwt.JWTClaims

	claims.FromMapClaims(parsed.Claims)

	validationErr, ok := err.(*jwt.ValidationError)
	if !ok {
		return nil, errorsx.WithStack(fosite.ErrServerError.WithDebugf("Unknown error: %s", err))
	}

	switch validationErr.Errors {
	case jwt.ValidationErrorUnverifiable:
		return nil, errorsx.WithStack(fosite.ErrServerError.WithHintf("Server error: %s", err))
	default:
		cause := types.ErrorInvalidTokenRequest{
			Subject: map[string]string{
				"issuer":  claims.Issuer,
				"subject": claims.Subject,
			},
		}

		fositeErr := fosite.ErrInvalidRequest.WithHintf("Invalid subject token: %s", err).WithWrap(cause)
		stackErr := errorsx.WithStack(fositeErr)

		return nil, stackErr
	}
}

func (s *TokenExchangeHandler) getSubjectClaims(ctx context.Context, token string) (*jwt.JWTClaims, error) {
	ctx, span := s.tracer.Start(ctx, "getSubjectClaims")

	defer span.End()

	validated, err := s.validateJWT(ctx, token)

	if err != nil {
		return nil, err
	}

	var claims jwt.JWTClaims

	claims.FromMapClaims(validated.Claims)

	return &claims, nil
}

func (s *TokenExchangeHandler) getMappedSubjectClaims(ctx context.Context, claims *jwt.JWTClaims) (jwt.JWTClaimsContainer, error) {
	ctx, span := s.tracer.Start(ctx, "getMappedSubjectClaims")

	defer span.End()

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
	ctx, span := s.tracer.Start(ctx, "HandleTokenEndpointRequest")

	defer span.End()

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

	// Set JWT claims as attributes if we have them
	span.SetAttributes(
		attribute.String(
			"subject_claims.iss",
			claims.Issuer,
		),
		attribute.String(
			"subject_claims.sub",
			claims.Subject,
		),
	)

	mappedClaims, err := s.getMappedSubjectClaims(ctx, claims)
	if err != nil {
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("error mapping claims: %s", err))
	}

	mappedSubjectClaim := *claims

	if sub, ok := mappedClaims.ToMapClaims()["sub"]; ok && sub != nil {
		if substr, ok := sub.(string); ok {
			mappedSubjectClaim.Subject = substr
		}
	}

	userInfoSvc := s.config.GetUserInfoStrategy(ctx)

	txManager, ok := userInfoSvc.(storage.TransactionManager)
	if !ok {
		return errorsx.WithStack(fosite.ErrServerError.WithHint("unable to find transaction manager"))
	}

	dbCtx, err := txManager.BeginContext(ctx)
	if err != nil {
		return errorsx.WithStack(fosite.ErrServerError.WithHint("could not start transaction"))
	}

	userInfo, err := s.populateUserInfo(dbCtx, &mappedSubjectClaim)
	if err != nil {
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("unable to populate user info: %s", err))
	}

	if subOverride, ok := mappedClaims.ToMapClaims()["identity-api.infratographer.com/sub"]; ok && subOverride != nil && subOverride.(string) != "" {
		issHash := sha256.Sum256([]byte(userInfo.Issuer))

		digest := base64.RawURLEncoding.EncodeToString(issHash[:])

		customPrefixedID, err := gidx.Parse(fmt.Sprintf("%s-%s-%s", types.IdentityUserIDPrefix, digest, subOverride))

		if err != nil {
			return errorsx.WithStack(fosite.ErrServerError.WithHintf("could not parse overridden subject to prefixed id: %s", err))
		}

		userInfo.ID = customPrefixedID
	}

	userInfo, err = userInfoSvc.StoreUserInfo(dbCtx, userInfo)

	if err != nil {
		rbErr := txManager.RollbackContext(dbCtx)
		return errorsx.WithStack(fosite.ErrInvalidRequest.WithHintf("unable to store user info: %s / rollback error: %s", err, rbErr))
	}

	err = txManager.CommitContext(dbCtx)

	if err != nil {
		return errorsx.WithStack(fosite.ErrServerError.WithHintf("could not commit user info: %s", err))
	}

	var newClaims jwt.JWTClaims
	newClaims.Subject = userInfo.ID.String()
	newClaims.Issuer = s.config.GetAccessTokenIssuer(ctx)

	for k, v := range mappedClaims.ToMapClaims() {
		if k != "identity-api.infratographer.com/sub" {
			newClaims.Add(k, v)
		}
	}

	expiry := time.Now().Add(s.config.GetAccessTokenLifespan(ctx))
	expiryMap := map[fosite.TokenType]time.Time{
		fosite.AccessToken: expiry,
	}

	var clientID *string

	maybeClientID := requester.GetClient().GetID()
	if len(maybeClientID) > 0 {
		clientID = &maybeClientID
	}

	newClaims.Add(ClaimClientID, clientID)

	kid := s.config.GetSigningKey(ctx).KeyID

	headers := jwt.Headers{}
	headers.Add("kid", kid)

	span.SetAttributes(
		attribute.String(
			"jwt_headers.kid",
			kid,
		),
		attribute.String(
			"jwt_claims.sub",
			newClaims.Subject,
		),
		attribute.String(
			"jwt_claims.exp",
			expiry.Format(time.RFC3339),
		),
	)

	var session *oauth2.JWTSession

	reqSess := requester.GetSession()
	if reqSess == nil {
		session = &oauth2.JWTSession{}
	} else {
		s, ok := reqSess.(*oauth2.JWTSession)
		if !ok {
			return errorsx.WithStack(fosite.ErrServerError.WithHint("requester session is not a jwt session"))
		}

		session = s
	}

	session.JWTHeader = &headers
	session.JWTClaims = &newClaims
	session.ExpiresAt = expiryMap
	session.Subject = claims.Subject

	userInfoAud, err := url.JoinPath(newClaims.Issuer, "userinfo")
	if err != nil {
		return errorsx.WithStack(fosite.ErrServerError.WithHintf("failed to build userinfo audience: %s", err))
	}

	requester.GrantAudience(userInfoAud)
	requester.SetSession(session)

	return nil
}

// PopulateTokenEndpointResponse populates the response with a token.
func (s *TokenExchangeHandler) PopulateTokenEndpointResponse(ctx context.Context, requester fosite.AccessRequester, responder fosite.AccessResponder) error {
	ctx, span := s.tracer.Start(ctx, "PopulateTokenEndpointResponse")

	defer span.End()

	if !s.CanHandleTokenEndpointRequest(ctx, requester) {
		return errorsx.WithStack(fosite.ErrUnknownRequest)
	}

	token, _, err := s.accessTokenStrategy.GenerateAccessToken(ctx, requester)
	if err != nil {
		return err
	}

	responder.SetAccessToken(token)
	responder.SetExtra(responseIssuedTokenType, TokenTypeJWT)
	responder.SetTokenType(fosite.BearerAccessToken)
	responder.SetExpiresIn(s.config.GetAccessTokenLifespan(ctx))

	return nil
}

// CanSkipClientAuth always returns true, as client auth is not required for token exchange.
func (s *TokenExchangeHandler) CanSkipClientAuth(_ context.Context, _ fosite.AccessRequester) bool {
	return true
}

// CanHandleTokenEndpointRequest returns true if the grant type is token exchange.
func (s *TokenExchangeHandler) CanHandleTokenEndpointRequest(_ context.Context, requester fosite.AccessRequester) bool {
	return requester.GetGrantTypes().ExactOne(GrantTypeTokenExchange)
}

func (s *TokenExchangeHandler) populateUserInfo(ctx context.Context, claims *jwt.JWTClaims) (types.UserInfo, error) {
	ctx, span := s.tracer.Start(ctx, "populateUserInfo")

	defer span.End()

	userInfoSvc := s.config.GetUserInfoStrategy(ctx)
	userInfo, err := userInfoSvc.LookupUserInfoByClaims(ctx, claims.Issuer, claims.Subject)

	if err != nil {
		// We can handle ErrUserInfoNotFound by hitting the
		// issuers userinfo endpoint, but if some other error
		// came back bail.
		if !errors.Is(err, types.ErrUserInfoNotFound) {
			fmt.Println("couldn't find issuer in lookup")
			return types.UserInfo{}, err
		}
	} else {
		return userInfo, nil
	}

	claimsMap := claims.ToMap()

	userInfo, err = userInfoSvc.ParseUserInfoFromClaims(claimsMap)
	if err != nil {
		fmt.Println("failed to fetch userinfo")
		return types.UserInfo{}, err
	}

	return userInfo, nil
}
