package fositex

import (
	"context"

	"github.com/ory/fosite"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/viperx"
	"gopkg.in/square/go-jose.v2"
)

const (
	PrivateKeyTypePublic    PrivateKeyType = "public"
	PrivateKeyTypeSymmetric PrivateKeyType = "symmetric"
)

// Issuer represents a configurable JWT issuer.
type Issuer struct {
	Name    string
	JWKSURI string
}

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
	SubjectTokenIssuers []Issuer
	Secret              string
	// When configuring an OAuth provider, the first private key will be used to sign
	// JWTs.
	PrivateKeys []PrivateKey
}

// IssuerJWKSURIStrategy represents a strategy for getting the JWKS URI for a given issuer.
type IssuerJWKSURIStrategy interface {
	GetIssuerJWKSURI(ctx context.Context, iss string) (string, error)
}

// IssuerJWKSURIStrategyProvider represents a provider for a IssuerJWKSURIStrategy.
type IssuerJWKSURIStrategyProvider interface {
	GetIssuerJWKSURIStrategy(ctx context.Context) IssuerJWKSURIStrategy
}

// SigningKeyProvider represents a provider of a signing key.
type SigningKeyProvider interface {
	GetSigningKey(ctx context.Context) *jose.JSONWebKey
}

// SigningJWKSProvider represents a provider of a valid signing JWKS.
type SigningJWKSProvider interface {
	GetSigningJWKS(ctx context.Context) *jose.JSONWebKeySet
}

// OAuth2Configurator represents an OAuth2 configuration.
type OAuth2Configurator interface {
	fosite.Configurator
	IssuerJWKSURIStrategyProvider
	SigningKeyProvider
	SigningJWKSProvider
}

// OAuth2Config represents a Fosite OAuth 2.0 provider configuration.
type OAuth2Config struct {
	*fosite.Config
	SigningKey            *jose.JSONWebKey
	SigningJWKS           *jose.JSONWebKeySet
	IssuerJWKSURIStrategy IssuerJWKSURIStrategy
}

// GetIssuerJWKSURIStrategy returns the config's IssuerJWKSURIStrategy.
func (c *OAuth2Config) GetIssuerJWKSURIStrategy(ctx context.Context) IssuerJWKSURIStrategy {
	return c.IssuerJWKSURIStrategy
}

// GetSigningKey returns the config's signing key.
func (c *OAuth2Config) GetSigningKey(ctx context.Context) *jose.JSONWebKey {
	return c.SigningKey
}

// GetSigningKey returns the config's signing JWKS. This includes private keys.
func (c *OAuth2Config) GetSigningJWKS(ctx context.Context) *jose.JSONWebKeySet {
	return c.SigningJWKS
}

// MustViperFlags sets the flags needed for Fosite to work.
func MustViperFlags(v *viper.Viper, flags *pflag.FlagSet, defaultListen string) {
	flags.String("issuer", "", "oauth token issuer")
	viperx.MustBindFlag(v, "oauth.issuer", flags.Lookup("issuer"))
	flags.String("private-key", "", "private key file")
	viperx.MustBindFlag(v, "oauth.privatekeyfile", flags.Lookup("issuer"))
}
