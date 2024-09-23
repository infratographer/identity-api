package v1

import "go.infratographer.com/identity-api/internal/crdbx"

var _ crdbx.Paginator = ListGroupMembersParams{}

// GetCursor implements crdbx.Paginator returning the cursor.
func (p ListGroupMembersParams) GetCursor() *crdbx.Cursor {
	return p.Cursor
}

// GetLimit implements crdbx.Paginator returning requested limit.
func (p ListGroupMembersParams) GetLimit() int {
	if p.Limit == nil {
		return 0
	}

	return *p.Limit
}

// GetOnlyFields implements crdbx.Paginator setting the only permitted field to `id`.
func (p ListGroupMembersParams) GetOnlyFields() []string {
	return []string{"subject_id"}
}

// SetPagination sets the pagination on the provided collection.
func (p ListGroupMembersParams) SetPagination(collection *GroupMemberCollection) error {
	collection.Pagination.Limit = crdbx.Limit(p.GetLimit())

	if count := len(collection.MemberIDs); count != 0 && count == collection.Pagination.Limit {
		last := collection.MemberIDs[count-1]

		cursor, err := crdbx.NewCursor(
			"subject_id", last.String(),
		)
		if err != nil {
			return err
		}

		collection.Pagination.Next = cursor
	}

	return nil
}
