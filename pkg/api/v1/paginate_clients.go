package v1

import "go.infratographer.com/identity-api/internal/crdbx"

var _ crdbx.Paginator = GetOwnerOAuthClientsParams{}

// GetCursor implements crdbx.Paginator returning the cursor.
func (p GetOwnerOAuthClientsParams) GetCursor() *crdbx.Cursor {
	return p.Cursor
}

// GetLimit implements crdbx.Paginator returning requested limit.
func (p GetOwnerOAuthClientsParams) GetLimit() int {
	if p.Limit == nil {
		return 0
	}

	return *p.Limit
}

// GetOnlyFields implements crdbx.Paginator setting the only permitted field to `id`.
func (p GetOwnerOAuthClientsParams) GetOnlyFields() []string {
	return []string{"id"}
}

// SetPagination sets the pagination on the provided collection.
func (p GetOwnerOAuthClientsParams) SetPagination(collection *OAuthClientCollection) error {
	collection.Pagination.Limit = crdbx.Limit(p.GetLimit())

	if count := len(collection.Clients); count != 0 && count == collection.Pagination.Limit {
		cursor, err := crdbx.NewCursor("id", collection.Clients[count-1].ID.String())
		if err != nil {
			return err
		}

		collection.Pagination.Next = cursor
	}

	return nil
}
