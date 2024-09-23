package v1

import "go.infratographer.com/identity-api/internal/crdbx"

var _ crdbx.Paginator = ListUserGroupsParams{}

// GetCursor implements crdbx.Paginator returning the cursor.
func (p ListUserGroupsParams) GetCursor() *crdbx.Cursor {
	return p.Cursor
}

// GetLimit implements crdbx.Paginator returning requested limit.
func (p ListUserGroupsParams) GetLimit() int {
	if p.Limit == nil {
		return 0
	}

	return *p.Limit
}

// GetOnlyFields implements crdbx.Paginator setting the only permitted field to `id`.
func (p ListUserGroupsParams) GetOnlyFields() []string {
	return []string{"group_id"}
}

// SetPagination sets the pagination on the provided collection.
func (p ListUserGroupsParams) SetPagination(collection *GroupIDCollection) error {
	collection.Pagination.Limit = crdbx.Limit(p.GetLimit())

	if count := len(collection.GroupIDs); count != 0 && count == collection.Pagination.Limit {
		cursor, err := crdbx.NewCursor("group_id", collection.GroupIDs[count-1].String())
		if err != nil {
			return err
		}

		collection.Pagination.Next = cursor
	}

	return nil
}
