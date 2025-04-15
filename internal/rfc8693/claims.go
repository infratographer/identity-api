package rfc8693

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/ory/fosite/token/jwt"

	"go.infratographer.com/identity-api/internal/celutils"
	"go.infratographer.com/identity-api/internal/fositex"
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
		return nil, ErrMissingSub
	}

	if claims.Issuer == "" {
		return nil, ErrMissingIss
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

// ClaimConditionStrategy represents a strategy for evaluating claims conditions.
type ClaimConditionStrategy struct {
	issuerSvc types.IssuerService
}

// NewClaimConditionStrategy creates a ClaimConditionStrategy given an issuer service.
func NewClaimConditionStrategy(issuerSvc types.IssuerService) ClaimConditionStrategy {
	return ClaimConditionStrategy{
		issuerSvc: issuerSvc,
	}
}

// ClaimConditionStrategy implements fositex.ClaimConditionStrategy
var _ fositex.ClaimConditionStrategy = (*ClaimConditionStrategy)(nil)

// Eval evaluates the claims conditions for the given claims.
func (c ClaimConditionStrategy) Eval(ctx context.Context, claims *jwt.JWTClaims) (bool, error) {
	if claims.Issuer == "" {
		return false, ErrMissingIss
	}

	iss := claims.Issuer

	issuer, err := c.issuerSvc.GetIssuerByURI(ctx, iss)
	if err != nil {
		return false, err
	}

	if issuer.ClaimConditions == nil || issuer.ClaimConditions.AST() == nil {
		return true, nil
	}

	inputEnv := map[string]any{
		celutils.CELVariableClaims: claims.ToMapClaims(),
	}

	res, err := celutils.Eval(issuer.ClaimConditions.AST(), inputEnv)
	if err != nil {
		return false, err
	}

	result, ok := res.Value().(bool)
	if !ok {
		return false, fmt.Errorf("%w: unexpected type for claim condition result: %T", ErrInvalidClaimCondition, res.Value())
	}

	return result, nil
}
