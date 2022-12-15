package rfc8693

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/cel-go/cel"
	"github.com/ory/fosite/token/jwt"
)

const (
	celVariableClaims    = "claims"
	celVariableSubSHA256 = "subSHA256"
)

func parseCEL(input string) (cel.Program, error) {
	env, err := cel.NewEnv(
		cel.Variable(celVariableClaims, cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable(celVariableSubSHA256, cel.StringType),
	)

	if err != nil {
		return nil, err
	}

	ast, issues := env.Compile(input)
	if err := issues.Err(); err != nil {
		wrapped := ErrorCELParse{
			inner: err,
		}

		return nil, &wrapped
	}

	prog, err := env.Program(ast)
	if err != nil {
		wrapped := ErrorCELParse{
			inner: err,
		}

		return nil, &wrapped
	}

	return prog, nil
}

// ClaimMappingStrategy represents a mapping from external identity claims to DMV claims.
type ClaimMappingStrategy struct {
	mappings map[string]cel.Program
}

// NewClaimMappingStrategy creates a ClaimMappingStrategy given a mapping of desired DMV claims to CEL expressions.
func NewClaimMappingStrategy(mappingExprs map[string]string) (ClaimMappingStrategy, error) {
	mappings := make(map[string]cel.Program, len(mappingExprs))

	for k, e := range mappingExprs {
		prog, err := parseCEL(e)
		if err != nil {
			return ClaimMappingStrategy{}, err
		}

		mappings[k] = prog
	}

	out := ClaimMappingStrategy{
		mappings: mappings,
	}

	return out, nil
}

// MapClaims consumes a set of JWT claims and produces a new set of mapped claims.
func (m ClaimMappingStrategy) MapClaims(claims *jwt.JWTClaims) (jwt.JWTClaimsContainer, error) {
	inputMap := claims.ToMapClaims()
	outputMap := make(map[string]any, len(m.mappings))

	subSHA256Bytes := sha256.Sum256([]byte(claims.Subject))
	subSHA256 := hex.EncodeToString(subSHA256Bytes[0:])

	inputEnv := map[string]any{
		celVariableClaims:    inputMap,
		celVariableSubSHA256: subSHA256,
	}

	for k, prog := range m.mappings {
		out, _, err := prog.Eval(inputEnv)

		if err != nil {
			wrapped := ErrorCELEval{
				inner: err,
			}

			return nil, &wrapped
		}

		outputMap[k] = out.Value()
	}

	var outputClaims jwt.JWTClaims

	outputClaims.FromMap(outputMap)

	return &outputClaims, nil
}
