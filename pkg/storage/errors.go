package storage

import "fmt"

// ErrorMissingEngineType represents an error where no engine type was provided.
var ErrorMissingEngineType = fmt.Errorf("missing engine type")

// ErrorUnknownEngineType represents an error where an invalid engine type was provided.
type ErrorUnknownEngineType struct {
	engineType EngineType
}

func (e *ErrorUnknownEngineType) Error() string {
	return fmt.Sprintf("unknown engine type '%s'", e.engineType)
}
