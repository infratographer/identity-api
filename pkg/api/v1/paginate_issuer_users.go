package v1

import "go.infratographer.com/identity-api/internal/crdbx"

var _ crdbx.Paginator = GetIssuerUsersParams{}

// GetCursor implements crdbx.Paginator returning the cursor.
func (p GetIssuerUsersParams) GetCursor() *crdbx.Cursor {
	return p.Cursor
}

// GetLimit implements crdbx.Paginator returning requested limit.
func (p GetIssuerUsersParams) GetLimit() int {
	if p.Limit == nil {
		return 0
	}

	return *p.Limit
}

// GetOnlyFields implements crdbx.Paginator setting the only permitted field to `id`.
func (p GetIssuerUsersParams) GetOnlyFields() []string {
	return []string{"id"}
}

// SetPagination sets the pagination on the provided collection.
func (p GetIssuerUsersParams) SetPagination(collection *UserCollection) error {
	collection.Pagination.Limit = crdbx.Limit(p.GetLimit())

	if count := len(collection.Users); count != 0 && count == collection.Pagination.Limit {
		cursor, err := crdbx.NewCursor("id", collection.Users[count-1].ID.String())
		if err != nil {
			return err
		}

		collection.Pagination.Next = cursor
	}

	return nil
}
