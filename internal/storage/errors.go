package storage

import "fmt"

// ErrorMissingEngineType represents an error where no engine type was provided.
var ErrorMissingEngineType = fmt.Errorf("missing engine type")

// ErrorMissingContextTx represents an error where no context transaction was provided.
var ErrorMissingContextTx = fmt.Errorf("no transaction provided in context")

// ErrorInvalidContextTx represents an error where the given context transaction is of the wrong type.
var ErrorInvalidContextTx = fmt.Errorf("invalid type for transaction context")

// ErrorUnsupportedEngineType represents an error where an invalid engine type was provided.
type ErrorUnsupportedEngineType struct {
	engineType EngineType
}

func (e *ErrorUnsupportedEngineType) Error() string {
	return fmt.Sprintf("unsupported engine type '%s'", e.engineType)
}
