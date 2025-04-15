package fositex

import (
	"context"

	jose "github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite"
	"github.com/ory/fosite/token/jwt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/viperx"

	"go.infratographer.com/identity-api/internal/types"
)

const (
	// PrivateKeyTypePublic represents a public key type.
	PrivateKeyTypePublic PrivateKeyType = "public"
	// PrivateKeyTypeSymmetric represents a symmetric key type.
	PrivateKeyTypeSymmetric PrivateKeyType = "symmetric"
)

// PrivateKeyType represents a key type (public or symmetric)
type PrivateKeyType string

// PrivateKey represents a path to a private key on disk with a given key ID.
type PrivateKey struct {
	KeyID     string
	Algorithm jose.SignatureAlgorithm
	Path      string
}

// Config represents an application config section for Fosite.
type Config struct {
	Issuer              string
	AccessTokenLifespan int
	Secret              string
	// When configuring an OAuth provider, the first private key will be used to sign
	// JWTs.
	PrivateKeys []PrivateKey
}

// IssuerJWKSURIProvider represents a provider for the JWKS URI for a given issuer.
type IssuerJWKSURIProvider interface {
	GetIssuerJWKSURI(ctx context.Context, iss string) (string, error)
}

// SigningKeyProvider represents a provider of a signing key.
type SigningKeyProvider interface {
	GetSigningKey(ctx context.Context) *jose.JSONWebKey
}

// SigningJWKSProvider represents a provider of a valid signing JWKS.
type SigningJWKSProvider interface {
	GetSigningJWKS(ctx context.Context) *jose.JSONWebKeySet
}

// ClaimMappingStrategy represents a strategy for mapping token claims to other claims.
type ClaimMappingStrategy interface {
	MapClaims(ctx context.Context, claims *jwt.JWTClaims) (jwt.JWTClaimsContainer, error)
}

// ClaimMappingStrategyProvider represents a provider of a claims mapping strategy.
type ClaimMappingStrategyProvider interface {
	GetClaimMappingStrategy(ctx context.Context) ClaimMappingStrategy
}

// ClaimConditionStrategyProvider represents a provider of a claims condition eval strategy.
type ClaimConditionStrategyProvider interface {
	GetClaimConditionStrategy(ctx context.Context) ClaimConditionStrategy
}

// ClaimConditionStrategy represents a strategy for evaluating claims conditions.
type ClaimConditionStrategy interface {
	Eval(ctx context.Context, claims *jwt.JWTClaims) (bool, error)
}

// UserInfoStrategy persists user information in the storage backend.
type UserInfoStrategy interface {
	types.UserInfoService
}

// UserInfoAudienceProvider returns the user info audience to attach to tokens
type UserInfoAudienceProvider interface {
	// GetUserInfoAudience returns the audience for the identity-api issuer
	GetUserInfoAudience() string
}

// UserInfoStrategyProvider represents the provider of the UserInfoStrategy.
type UserInfoStrategyProvider interface {
	GetUserInfoStrategy(ctx context.Context) UserInfoStrategy
}

// OAuth2Configurator represents an OAuth2 configuration.
type OAuth2Configurator interface {
	fosite.Configurator
	SigningKeyProvider
	SigningJWKSProvider
	ClaimMappingStrategyProvider
	ClaimConditionStrategyProvider
	UserInfoStrategyProvider
	GetIssuerJWKSURIProvider(ctx context.Context) IssuerJWKSURIProvider
}

// OAuth2Config represents a Fosite OAuth 2.0 provider configuration.
type OAuth2Config struct {
	*fosite.Config
	SigningKey  *jose.JSONWebKey
	SigningJWKS *jose.JSONWebKeySet

	ClaimMappingStrategy   ClaimMappingStrategy
	ClaimConditionStrategy ClaimConditionStrategy
	UserInfoStrategy       UserInfoStrategy

	IssuerJWKSURIProvider IssuerJWKSURIProvider
	userInfoAudience      string
}

// GetIssuerJWKSURIProvider returns the config's IssuerJWKSURIProvider.
func (c *OAuth2Config) GetIssuerJWKSURIProvider(_ context.Context) IssuerJWKSURIProvider {
	return c.IssuerJWKSURIProvider
}

// GetSigningKey returns the config's signing key.
func (c *OAuth2Config) GetSigningKey(_ context.Context) *jose.JSONWebKey {
	return c.SigningKey
}

// GetSigningJWKS returns the config's signing JWKS. This includes private keys.
func (c *OAuth2Config) GetSigningJWKS(_ context.Context) *jose.JSONWebKeySet {
	return c.SigningJWKS
}

// GetClaimMappingStrategy returns the config's claims mapping strategy.
func (c *OAuth2Config) GetClaimMappingStrategy(_ context.Context) ClaimMappingStrategy {
	return c.ClaimMappingStrategy
}

// GetClaimConditionStrategy returns the config's claim condition strategy.
func (c *OAuth2Config) GetClaimConditionStrategy(_ context.Context) ClaimConditionStrategy {
	return c.ClaimConditionStrategy
}

// GetUserInfoStrategy returns the config's user info store strategy.
func (c *OAuth2Config) GetUserInfoStrategy(_ context.Context) UserInfoStrategy {
	return c.UserInfoStrategy
}

// GetUserInfoAudience returns this services userinfo audience.
func (c *OAuth2Config) GetUserInfoAudience() string {
	return c.userInfoAudience
}

// MustViperFlags sets the flags needed for Fosite to work.
func MustViperFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("issuer", "", "oauth token issuer")
	viperx.MustBindFlag(v, "oauth.issuer", flags.Lookup("issuer"))
	flags.String("private-key", "", "private key file")
	viperx.MustBindFlag(v, "oauth.privatekeyfile", flags.Lookup("issuer"))
}
