package v1

import "go.infratographer.com/identity-api/internal/crdbx"

var _ crdbx.Paginator = ListGroupsParams{}

// GetCursor implements crdbx.Paginator returning the cursor.
func (p ListGroupsParams) GetCursor() *crdbx.Cursor {
	return p.Cursor
}

// GetLimit implements crdbx.Paginator returning requested limit.
func (p ListGroupsParams) GetLimit() int {
	if p.Limit == nil {
		return 0
	}

	return *p.Limit
}

// GetOnlyFields implements crdbx.Paginator setting the only permitted field to `id`.
func (p ListGroupsParams) GetOnlyFields() []string {
	return []string{"id"}
}

// SetPagination sets the pagination on the provided collection.
func (p ListGroupsParams) SetPagination(collection *GroupCollection) error {
	collection.Pagination.Limit = crdbx.Limit(p.GetLimit())

	if count := len(collection.Groups); count != 0 && count == collection.Pagination.Limit {
		cursor, err := crdbx.NewCursor("id", collection.Groups[count-1].ID.String())
		if err != nil {
			return err
		}

		collection.Pagination.Next = cursor
	}

	return nil
}
