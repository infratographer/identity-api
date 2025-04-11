package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"go.infratographer.com/x/gidx"

	"go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/types"
)

var groupCols = struct {
	ID          string
	OwnerID     string
	Name        string
	Description string
}{
	ID:          "id",
	OwnerID:     "owner_id",
	Name:        "name",
	Description: "description",
}

var groupMemberCols = struct {
	GroupID   string
	SubjectID string
}{
	GroupID:   "group_id",
	SubjectID: "subject_id",
}

var groupColsStr = strings.Join([]string{
	groupCols.ID, groupCols.OwnerID,
	groupCols.Name, groupCols.Description,
}, ", ")

const (
	membersTable = "group_members"
	groupsTable  = "groups"
)

type groupService struct {
	db *sql.DB
}

type groupServiceOpt func(*groupService)

func newGroupService(db *sql.DB, opts ...groupServiceOpt) (*groupService, error) {
	s := &groupService{
		db: db,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (gs *groupService) CreateGroup(ctx context.Context, group types.Group) (*types.Group, error) {
	if err := gs.insertGroup(ctx, group); err != nil {
		if isPQDuplicateKeyError(err) {
			return nil, types.ErrGroupExists
		}

		return nil, err
	}

	g, err := gs.fetchGroupByID(ctx, group.ID)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (gs *groupService) GetGroupByID(ctx context.Context, id gidx.PrefixedID) (*types.Group, error) {
	return gs.fetchGroupByID(ctx, id)
}

func (gs *groupService) insertGroup(ctx context.Context, group types.Group) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	if group.Name == "" {
		return types.ErrGroupNameEmpty
	}

	cols := []string{
		groupCols.ID,
		groupCols.OwnerID,
		groupCols.Name,
		groupCols.Description,
	}

	q := fmt.Sprintf(
		"INSERT INTO groups (%s) VALUES ($1, $2, $3, $4)",
		strings.Join(cols, ", "),
	)

	_, err = tx.ExecContext(
		ctx, q,
		group.ID, group.OwnerID, group.Name, group.Description,
	)

	return err
}

func (gs *groupService) fetchGroupByID(ctx context.Context, id gidx.PrefixedID) (*types.Group, error) {
	q := fmt.Sprintf(
		"SELECT %s FROM groups WHERE %s = $1",
		groupColsStr, groupCols.ID,
	)

	var ex func(ctx context.Context, query string, args ...any) *sql.Row

	tx, err := getContextTx(ctx)
	switch err {
	case nil:
		ex = tx.QueryRowContext
	case ErrorMissingContextTx:
		ex = gs.db.QueryRowContext
	default:
		return nil, err
	}

	row := ex(ctx, q, id)

	return gs.scanGroup(row)
}

func (gs *groupService) scanGroup(row *sql.Row) (*types.Group, error) {
	var g types.Group

	err := row.Scan(
		&g.ID, &g.OwnerID, &g.Name, &g.Description,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, types.ErrGroupNotFound
	case err != nil:
		return nil, err
	default:
	}

	return &g, nil
}

func (gs *groupService) ListGroupsByOwner(ctx context.Context, ownerID gidx.PrefixedID, pagination crdbx.Paginator) (types.Groups, error) {
	paginate := crdbx.Paginate(pagination, crdbx.ContextAsOfSystemTime(ctx, "-1m"))

	q := fmt.Sprintf(
		"SELECT %s FROM groups %s WHERE %s = $1 %s %s %s",
		groupColsStr, paginate.AsOfSystemTime(), groupCols.OwnerID,
		paginate.AndWhere(2), //nolint:mnd
		paginate.OrderClause(),
		paginate.LimitClause(),
	)

	rows, err := gs.db.QueryContext(ctx, q, paginate.Values(ownerID)...)
	if err != nil {
		return nil, err
	}

	defer rows.Close() //nolint:errcheck

	var groups types.Groups

	for rows.Next() {
		g := &types.Group{}

		if err := rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.Description); err != nil {
			return nil, err
		}

		groups = append(groups, g)
	}

	return groups, nil
}

func (gs *groupService) UpdateGroup(ctx context.Context, id gidx.PrefixedID, updates types.GroupUpdate) (*types.Group, error) {
	tx, err := getContextTx(ctx)
	if err != nil {
		return nil, err
	}

	current, err := gs.fetchGroupByID(ctx, id)
	if err != nil {
		return nil, err
	}

	incoming := *current

	if updates.Name != nil && *updates.Name != "" {
		incoming.Name = *updates.Name
	}

	if updates.Description != nil {
		incoming.Description = *updates.Description
	}

	q := fmt.Sprintf(
		"UPDATE groups SET (%s, %s) = ($1, $2) WHERE %s = $3",
		groupCols.Name, groupCols.Description, groupCols.ID,
	)

	if _, err := tx.ExecContext(ctx, q, incoming.Name, incoming.Description, incoming.ID); err != nil {
		if isPQDuplicateKeyError(err) {
			return nil, types.ErrGroupExists
		}

		return nil, err
	}

	return &incoming, nil
}

func (gs *groupService) DeleteGroup(ctx context.Context, id gidx.PrefixedID) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	if _, err := gs.fetchGroupByID(ctx, id); err != nil {
		return err
	}

	q := fmt.Sprintf(
		"DELETE FROM groups WHERE %s = $1",
		groupCols.ID,
	)

	_, err = tx.ExecContext(ctx, q, id)

	return err
}

func (gs *groupService) AddGroupMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) error {
	if len(subjects) == 0 {
		return nil
	}

	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	if _, err := gs.fetchGroupByID(ctx, groupID); err != nil {
		return err
	}

	vals := make([]string, 0, len(subjects))
	params := make([]any, 0, len(subjects)+1)
	params = append(params, groupID)

	const placeholderOffset = 2

	for i, subj := range subjects {
		vals = append(vals, fmt.Sprintf("($1, $%d)", i+placeholderOffset))
		params = append(params, subj)
	}

	q := fmt.Sprintf(
		"UPSERT INTO group_members (%s, %s) VALUES %s",
		groupMemberCols.GroupID, groupMemberCols.SubjectID,
		strings.Join(vals, ", "),
	)

	_, err = tx.ExecContext(ctx, q, params...)
	if err != nil {
		fmt.Println(err.Error())
	}

	return err
}

func (gs *groupService) ListGroupMembers(ctx context.Context, groupID gidx.PrefixedID, pagination crdbx.Paginator) ([]gidx.PrefixedID, error) {
	if _, err := gs.fetchGroupByID(ctx, groupID); err != nil {
		return nil, err
	}

	var ex func() (*sql.Rows, error)

	if pagination != nil {
		paginate := crdbx.Paginate(pagination, crdbx.ContextAsOfSystemTime(ctx, "-1m"))

		q := fmt.Sprintf(
			"SELECT %s FROM %s %s WHERE %s = $1 %s %s %s",
			groupMemberCols.SubjectID, membersTable,
			paginate.AsOfSystemTime(), groupMemberCols.GroupID,
			paginate.AndWhere(2), //nolint:mnd
			paginate.OrderClause(),
			paginate.LimitClause(),
		)

		ex = func() (*sql.Rows, error) {
			return gs.db.QueryContext(ctx, q, paginate.Values(groupID)...)
		}
	} else {
		q := fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s = $1",
			groupMemberCols.SubjectID, membersTable, groupMemberCols.GroupID,
		)

		ex = func() (*sql.Rows, error) {
			return gs.db.QueryContext(ctx, q, groupID)
		}
	}

	rows, err := ex()
	if err != nil {
		return nil, err
	}

	defer rows.Close() //nolint:errcheck

	var members []gidx.PrefixedID

	for rows.Next() {
		var member gidx.PrefixedID

		if err := rows.Scan(&member); err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return members, nil
}

func (gs *groupService) RemoveGroupMember(ctx context.Context, groupID gidx.PrefixedID, subject gidx.PrefixedID) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	if _, err := gs.fetchGroupByID(ctx, groupID); err != nil {
		return err
	}

	q := fmt.Sprintf(
		"DELETE FROM group_members WHERE %s = $1 AND %s = $2",
		groupMemberCols.GroupID, groupMemberCols.SubjectID,
	)

	res, err := tx.ExecContext(ctx, q, groupID, subject)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	} else if rowsAffected == 0 {
		return types.ErrGroupMemberNotFound
	}

	return err
}

func (gs *groupService) ReplaceGroupMembers(
	ctx context.Context, groupID gidx.PrefixedID, incoming ...gidx.PrefixedID,
) ([]gidx.PrefixedID, []gidx.PrefixedID, error) {
	tx, err := getContextTx(ctx)
	if err != nil {
		return nil, nil, err
	}

	current, err := gs.ListGroupMembers(ctx, groupID, nil)
	if err != nil {
		return nil, nil, err
	}

	valFn := func(x gidx.PrefixedID) string { return x.String() }
	add, rm := Diff(current, incoming, valFn)

	delq := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1 AND %s = ANY($2)",
		membersTable,
		groupMemberCols.GroupID, groupMemberCols.SubjectID,
	)

	_, err = tx.ExecContext(ctx, delq, groupID, pq.Array(rm))
	if err != nil {
		return nil, nil, err
	}

	if err := gs.AddGroupMembers(ctx, groupID, add...); err != nil {
		return nil, nil, err
	}

	return add, rm, nil
}

func (gs *groupService) ListGroupsBySubject(ctx context.Context, subject gidx.PrefixedID, pagination crdbx.Paginator) (types.Groups, error) {
	paginate := crdbx.Paginate(pagination, crdbx.ContextAsOfSystemTime(ctx, "-1m"))

	q := fmt.Sprintf(
		`SELECT %s FROM %s LEFT JOIN %s ON %s %s WHERE %s = $1 %s %s %s`,
		// SELECT
		strings.Join([]string{
			fmt.Sprintf("DISTINCT(%s.%s)", membersTable, groupMemberCols.GroupID),
			fmt.Sprintf("%s.%s", groupsTable, groupCols.Name),
			fmt.Sprintf("%s.%s", groupsTable, groupCols.Description),
			fmt.Sprintf("%s.%s", groupsTable, groupCols.OwnerID),
		}, ", "),
		// FROM
		membersTable,
		// LEFT JOIN
		groupsTable,
		// ON
		fmt.Sprintf(
			"%s.%s = %s.%s",
			groupsTable, groupCols.ID,
			membersTable, groupMemberCols.GroupID,
		),
		// as of system time
		paginate.AsOfSystemTime(),
		// WHERE
		fmt.Sprintf("%s.%s", membersTable, groupMemberCols.SubjectID),
		// Pagination
		paginate.AndWhere(2), //nolint:mnd
		paginate.OrderClause(),
		paginate.LimitClause(),
	)

	rows, err := gs.db.QueryContext(ctx, q, paginate.Values(subject)...)
	if err != nil {
		return nil, err
	}

	defer rows.Close() //nolint:errcheck

	var groups types.Groups

	for rows.Next() {
		g := &types.Group{}

		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.OwnerID); err != nil {
			return nil, err
		}

		groups = append(groups, g)
	}

	return groups, nil
}

func (gs *groupService) GroupMembersCount(ctx context.Context, groupID gidx.PrefixedID) (int, error) {
	q := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE %s = $1",
		membersTable, groupMemberCols.GroupID,
	)

	row := gs.db.QueryRowContext(ctx, q, groupID)

	var count int

	if err := row.Scan(&count); err != nil {
		return -1, err
	}

	return count, nil
}
