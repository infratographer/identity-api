package crdbx

import (
	"errors"
	"slices"
)

const (
	paginationLimitMax     = 100
	paginationLimitDefault = 10
)

// ErrInvalidPaginationCursor is returned when there's an error handling the cursor.
var ErrInvalidPaginationCursor = errors.New("invalid pagination cursor")

// Paginator provides the necessary details to paginate a database request.
type Paginator interface {
	GetCursor() *Cursor
	GetLimit() int
	GetOnlyFields() []string
}

var _ Paginator = Pagination{}

// Pagination define pagination query parameters.
type Pagination struct {
	Cursor     *Cursor
	Limit      int
	OnlyFields []string
}

// GetCursor implements [Paginator] returning the cursor
func (p Pagination) GetCursor() *Cursor {
	return p.Cursor
}

// GetLimit implements [Paginator] returning the effective limit.
func (p Pagination) GetLimit() int {
	return p.Limit
}

// GetOnlyFields implements [Paginator] returning the permitted fields.
func (p Pagination) GetOnlyFields() []string {
	return p.OnlyFields
}

// Limit accepts a requested limit, returning an acceptable limit.
func Limit(limit int) int {
	switch {
	case limit > paginationLimitMax:
		limit = paginationLimitMax
	case limit <= 0:
		limit = paginationLimitDefault
	}

	return limit
}

// Paginate handles setting the pagination for the request.
func Paginate(p Paginator, asOfSystemTime any) FormatValues {
	var values FormatValues

	values.limit = p.GetLimit()

	if asOfSystemTime != nil {
		values.asOfSystemTime = asOfSystemTime
	}

	onlyFields := p.GetOnlyFields()

	if len(onlyFields) == 0 {
		onlyFields = []string{"id"}
	}

	values.orderFields = onlyFields

	if cursor := p.GetCursor(); cursor != nil {
		cValues, err := cursor.Values()
		if err != nil {
			return FormatValues{}
		}

		// Ensure only permitted fields are defined.
		for key := range cValues {
			if !slices.Contains(onlyFields, key) {
				return FormatValues{}
			}
		}

		// Add clauses in original field order.
		for _, field := range onlyFields {
			if value := cValues.Get(field); value != "" {
				values.fields = append(values.fields, field)
				values.values = append(values.values, value)
			}
		}
	}

	return values
}
