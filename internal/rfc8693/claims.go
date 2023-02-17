package rfc8693

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/ory/fosite/token/jwt"

	"go.infratographer.com/identity-api/internal/celutils"
	"go.infratographer.com/identity-api/internal/types"
)

// ClaimMappingStrategy represents a mapping from external identity claims to identity-api claims.
type ClaimMappingStrategy struct {
	issuerSvc types.IssuerService
}

// NewClaimMappingStrategy creates a ClaimMappingStrategy given an issuer service.
func NewClaimMappingStrategy(issuerSvc types.IssuerService) ClaimMappingStrategy {
	out := ClaimMappingStrategy{
		issuerSvc: issuerSvc,
	}

	return out
}

// MapClaims consumes a set of JWT claims and produces a new set of mapped claims.
func (m ClaimMappingStrategy) MapClaims(ctx context.Context, claims *jwt.JWTClaims) (jwt.JWTClaimsContainer, error) {
	if claims.Subject == "" {
		return nil, ErrorMissingSub
	}

	if claims.Issuer == "" {
		return nil, ErrorMissingIss
	}

	iss := claims.Issuer

	issuer, err := m.issuerSvc.GetIssuerByURI(ctx, iss)
	if err != nil {
		return nil, err
	}

	inputMap := claims.ToMapClaims()
	outputMap := make(map[string]any, len(issuer.ClaimMappings))

	subSHA256Bytes := sha256.Sum256([]byte(claims.Subject))
	subSHA256 := hex.EncodeToString(subSHA256Bytes[0:])

	inputEnv := map[string]any{
		celutils.CELVariableClaims:    inputMap,
		celutils.CELVariableSubSHA256: subSHA256,
	}

	for k, v := range issuer.ClaimMappings {
		out, err := celutils.Eval(v, inputEnv)
		if err != nil {
			return nil, err
		}

		outputMap[k] = out.Value()
	}

	var outputClaims jwt.JWTClaims

	outputClaims.FromMap(outputMap)

	return &outputClaims, nil
}
