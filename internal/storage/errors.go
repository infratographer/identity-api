package storage

import (
	"fmt"

	"github.com/lib/pq"
)

var (
	// ErrorMissingContextTx represents an error where no context transaction was provided.
	ErrorMissingContextTx = fmt.Errorf("no transaction provided in context")
	// ErrorInvalidContextTx represents an error where the given context transaction is of the wrong type.
	ErrorInvalidContextTx = fmt.Errorf("invalid type for transaction context")
)

const (
	pqErrDuplicateKey = "23505"
)

func isPQError(err error) *pq.Error {
	if err == nil {
		return nil
	}

	pqerr, ok := err.(*pq.Error)
	if !ok {
		return nil
	}

	return pqerr
}

func isPQDuplicateKeyError(err error) bool {
	pqerr := isPQError(err)
	if pqerr == nil {
		return false
	}

	return pqerr.Code == pqErrDuplicateKey
}
