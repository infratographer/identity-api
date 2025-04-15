package storage

import (
	"fmt"
	"strings"

	"go.infratographer.com/identity-api/internal/types"
)

type colBinding struct {
	column string
	value  any
}

func colBindingsToParams(bindings []colBinding) (string, []any) {
	bindingStrs := make([]string, len(bindings))
	args := make([]any, len(bindings))

	for i, binding := range bindings {
		bindingStr := fmt.Sprintf("%s = $%d", binding.column, i+1)
		bindingStrs[i] = bindingStr
		args[i] = binding.value
	}

	bindingsStr := strings.Join(bindingStrs, ", ")

	return bindingsStr, args
}

func bindIfNotNil[T any](bindings []colBinding, column string, value *T) []colBinding {
	if value != nil {
		binding := colBinding{
			column: column,
			value:  *value,
		}

		return append(bindings, binding)
	}

	return bindings
}

func issuerUpdateToColBindings(update types.IssuerUpdate) ([]colBinding, error) {
	var bindings []colBinding

	bindings = bindIfNotNil(bindings, issuerCols.Name, update.Name)
	bindings = bindIfNotNil(bindings, issuerCols.URI, update.URI)
	bindings = bindIfNotNil(bindings, issuerCols.JWKSURI, update.JWKSURI)

	if update.ClaimMappings != nil {
		mappingRepr, err := update.ClaimMappings.MarshalJSON()
		if err != nil {
			return nil, err
		}

		mappingStr := string(mappingRepr)

		bindings = bindIfNotNil(bindings, issuerCols.Mappings, &mappingStr)
	}

	if update.ClaimConditions != nil {
		condRepr, err := update.ClaimConditions.MarshalJSON()
		if err != nil {
			return nil, err
		}

		condStr := string(condRepr)

		bindings = bindIfNotNil(bindings, issuerCols.Conditions, &condStr)
	}

	return bindings, nil
}

// withQualifier adds a qualifier to a column
// e.g. withQualifier([]string{"name"}, "ui") = []string{"ui.name"}
func withQualifier(items []string, qualifier string) []string {
	out := make([]string, len(items))
	for i, el := range items {
		out[i] = fmt.Sprintf("%s.%s", qualifier, el)
	}

	return out
}
