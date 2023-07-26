// Package fositex provides a wrapper around the fosite library to more easily
// use the parts that are relevant for us.
package fositex

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ory/fosite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/square/go-jose.v2"
)

const instrumentationName = "go.infratographer.com/identity-api/internal/fositex"

var (
	// ErrInvalidKey is returned when the key is not valid.
	ErrInvalidKey = fmt.Errorf("invalid key")
)

func readSymmetricKey(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func readAsymmetricKey[T crypto.Signer](path string) (T, error) {
	var empty T

	f, err := os.Open(path)
	if err != nil {
		return empty, err
	}

	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return empty, err
	}

	block, rest := pem.Decode(bytes)

	switch {
	case block == nil, block.Type != "PRIVATE KEY":
		return empty, fmt.Errorf("%w: invalid private key", ErrInvalidKey)
	case len(rest) > 0:
		return empty, fmt.Errorf("%w: extra data in private key", ErrInvalidKey)
	default:
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return empty, err
	}

	signer, ok := key.(T)
	if !ok {
		return empty, fmt.Errorf("%w: key is not a valid signing key", ErrInvalidKey)
	}

	return signer, nil
}

func readPrivateKey(key PrivateKey) (jose.JSONWebKey, error) {
	var (
		rawKey interface{}
		err    error
	)

	switch key.Algorithm {
	case jose.RS256, jose.RS384, jose.RS512:
		rawKey, err = readAsymmetricKey[*rsa.PrivateKey](key.Path)
	case jose.HS256, jose.HS384, jose.HS512:
		rawKey, err = readSymmetricKey(key.Path)
	default:
		return jose.JSONWebKey{}, fmt.Errorf("%w: unsupported private key type %s",
			ErrInvalidKey, key.Algorithm)
	}

	if err != nil {
		return jose.JSONWebKey{}, err
	}

	out := jose.JSONWebKey{
		Key:       rawKey,
		KeyID:     key.KeyID,
		Algorithm: string(key.Algorithm),
	}

	return out, nil
}

func parsePrivateKeys(keys []PrivateKey) (*jose.JSONWebKey, *jose.JSONWebKeySet, error) {
	if len(keys) == 0 {
		return nil, nil, fmt.Errorf("%w: no private keys provided", ErrInvalidKey)
	}

	first := keys[0]

	signingKey, err := readPrivateKey(first)
	if err != nil {
		return nil, nil, err
	}

	var jwks jose.JSONWebKeySet

	jwks.Keys = append(jwks.Keys, signingKey)

	for _, key := range keys[1:] {
		jwk, err := readPrivateKey(key)
		if err != nil {
			return nil, nil, err
		}

		jwks.Keys = append(jwks.Keys, jwk)
	}

	return &signingKey, &jwks, nil
}

// NewOAuth2Config builds a new OAuth2Config from the given Config.
func NewOAuth2Config(config Config) (*OAuth2Config, error) {
	signingKey, jwks, err := parsePrivateKeys(config.PrivateKeys)
	if err != nil {
		return nil, err
	}

	tokenLifespan := time.Second * time.Duration(config.AccessTokenLifespan)
	fositeConfig := &fosite.Config{
		AccessTokenIssuer:   config.Issuer,
		AccessTokenLifespan: tokenLifespan,
		GlobalSecret:        []byte(config.Secret),
	}

	userInfoAudience, err := url.JoinPath(config.Issuer, "userinfo")
	if err != nil {
		return nil, err
	}

	out := &OAuth2Config{
		Config:           fositeConfig,
		SigningKey:       signingKey,
		SigningJWKS:      jwks,
		userInfoAudience: userInfoAudience,
	}

	return out, nil
}

type instrumentedProvider struct {
	tracer trace.Tracer

	fosite.OAuth2Provider
}

func (p *instrumentedProvider) NewAccessRequest(ctx context.Context, req *http.Request, session fosite.Session) (fosite.AccessRequester, error) {
	ctx, span := p.tracer.Start(ctx, "fositex.NewAccessRequest")

	defer span.End()

	return p.OAuth2Provider.NewAccessRequest(ctx, req, session)
}

func (p *instrumentedProvider) NewAccessResponse(ctx context.Context, req fosite.AccessRequester) (fosite.AccessResponder, error) {
	ctx, span := p.tracer.Start(ctx, "fositex.NewAccessResponse")

	defer span.End()

	return p.OAuth2Provider.NewAccessResponse(ctx, req)
}

// NewOAuth2Provider creates a new fosite.OAuth2Provider.
// The configurator, store, and strategy are all passed to the factories
// and the resulting endpoint handlers are registered to the fosite.Config.
func NewOAuth2Provider(configurator *OAuth2Config, store interface{}, strategy interface{}, factories ...Factory) fosite.OAuth2Provider {
	config := configurator.Config
	storage := store.(fosite.Storage)

	f := fosite.NewOAuth2Provider(storage, config)

	for _, factory := range factories {
		res := factory(configurator, storage, strategy)

		if ah, ok := res.(fosite.AuthorizeEndpointHandler); ok {
			config.AuthorizeEndpointHandlers.Append(ah)
		}

		if th, ok := res.(fosite.TokenEndpointHandler); ok {
			config.TokenEndpointHandlers.Append(th)
		}

		if tv, ok := res.(fosite.TokenIntrospector); ok {
			config.TokenIntrospectionHandlers.Append(tv)
		}

		if rh, ok := res.(fosite.RevocationHandler); ok {
			config.RevocationHandlers.Append(rh)
		}

		if ph, ok := res.(fosite.PushedAuthorizeEndpointHandler); ok {
			config.PushedAuthorizeEndpointHandlers.Append(ph)
		}
	}

	tracer := otel.Tracer(instrumentationName)

	out := &instrumentedProvider{
		tracer:         tracer,
		OAuth2Provider: f,
	}

	return out
}

// Factory is a constructor which is used to create an OAuth2 endpoin handler.
// NewOAuth2Provider handles consuming the new struct and attaching it
// to the parts of the config that it implements.
type Factory func(config OAuth2Configurator, store any, strategy any) any
