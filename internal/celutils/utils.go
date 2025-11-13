package celutils

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
)

var (
	celEnv *cel.Env
)

func init() {
	env, err := cel.NewEnv(
		cel.Variable(CELVariableClaims, cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable(CELVariableSubSHA256, cel.StringType),
	)
	if err != nil {
		panic(err)
	}

	celEnv = env
}

// ParseCEL parses a CEL expression.
func ParseCEL(input string) (*cel.Ast, error) {
	ast, issues := celEnv.Compile(input)
	if err := issues.Err(); err != nil {
		wrapped := ErrorCELParse{
			inner: err,
		}

		return nil, &wrapped
	}

	return ast, nil
}

// Eval evaluates the given AST against the provided input environment.
func Eval(ast *cel.Ast, inputEnv map[string]any) (ref.Val, error) {
	prog, err := celEnv.Program(ast)
	if err != nil {
		wrapped := ErrorCELParse{
			inner: err,
		}

		return nil, &wrapped
	}

	val, _, err := prog.Eval(inputEnv)
	if err != nil {
		wrapped := ErrorCELEval{
			inner: err,
		}

		return nil, &wrapped
	}

	return val, nil
}
