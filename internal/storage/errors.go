package storage

import "fmt"

// ErrorMissingContextTx represents an error where no context transaction was provided.
var ErrorMissingContextTx = fmt.Errorf("no transaction provided in context")

// ErrorInvalidContextTx represents an error where the given context transaction is of the wrong type.
var ErrorInvalidContextTx = fmt.Errorf("invalid type for transaction context")
