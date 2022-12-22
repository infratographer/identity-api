package celutils

import "github.com/google/cel-go/cel"

// ParseCEL parses a CEL expression.
func ParseCEL(input string) (cel.Program, error) {
	env, err := cel.NewEnv(
		cel.Variable(CELVariableClaims, cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable(CELVariableSubSHA256, cel.StringType),
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
