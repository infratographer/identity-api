package fositex

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/ory/fosite"
	"gopkg.in/square/go-jose.v2"
	"io"
	"os"
	"time"
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
		return empty, fmt.Errorf("malformed private key")
	case len(rest) > 0:
		return empty, fmt.Errorf("extra data in private key")
	default:
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return empty, err
	}

	signer, ok := key.(T)
	if !ok {
		return empty, fmt.Errorf("key is not a valid signing key")
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
		return jose.JSONWebKey{}, fmt.Errorf("unsupported private key type %s", key.Algorithm)
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
		return nil, nil, fmt.Errorf("no private keys provided")
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

		var jwks jose.JSONWebKeySet

		jwks.Keys = append(jwks.Keys, jwk)
	}

	return &signingKey, &jwks, nil
}

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

	out := &OAuth2Config{
		Config:      fositeConfig,
		SigningKey:  signingKey,
		SigningJWKS: jwks,
	}

	return out, nil
}

func NewOAuth2Provider(config OAuth2Configurator, store fosite.Storage) fosite.OAuth2Provider {
	provider := fosite.NewOAuth2Provider(store, config)
	return provider
}
