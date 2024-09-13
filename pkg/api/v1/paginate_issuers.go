package v1

import "go.infratographer.com/identity-api/internal/crdbx"

var _ crdbx.Paginator = ListOwnerIssuersParams{}

// GetCursor implements crdbx.Paginator returning the cursor.
func (p ListOwnerIssuersParams) GetCursor() *crdbx.Cursor {
	return p.Cursor
}

// GetLimit implements crdbx.Paginator returning requested limit.
func (p ListOwnerIssuersParams) GetLimit() int {
	if p.Limit == nil {
		return 0
	}

	return *p.Limit
}

// GetOnlyFields implements crdbx.Paginator setting the only permitted field to `id`.
func (p ListOwnerIssuersParams) GetOnlyFields() []string {
	return []string{"id"}
}

// SetPagination sets the pagination on the provided collection.
func (p ListOwnerIssuersParams) SetPagination(collection *IssuerCollection) error {
	collection.Pagination.Limit = crdbx.Limit(p.GetLimit())

	if count := len(collection.Issuers); count != 0 && count == collection.Pagination.Limit {
		cursor, err := crdbx.NewCursor("id", collection.Issuers[count-1].ID.String())
		if err != nil {
			return err
		}

		collection.Pagination.Next = cursor
	}

	return nil
}
