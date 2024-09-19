package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/x/gidx"
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

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, q, id)
	case ErrorMissingContextTx:
		row = gs.db.QueryRowContext(ctx, q, id)
	default:
		return nil, err
	}

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

func (gs *groupService) ListGroups(ctx context.Context, ownerID gidx.PrefixedID, pagination crdbx.Paginator) (types.Groups, error) {
	paginate := crdbx.Paginate(pagination, crdbx.ContextAsOfSystemTime(ctx, "-1m"))

	q := fmt.Sprintf(
		"SELECT %s FROM groups %s WHERE %s = $1 %s %s %s",
		groupColsStr, paginate.AsOfSystemTime(), groupCols.OwnerID,
		paginate.AndWhere(2), //nolint:gomnd
		paginate.OrderClause(),
		paginate.LimitClause(),
	)

	rows, err := gs.db.QueryContext(ctx, q, paginate.Values(ownerID)...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

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

func (gs *groupService) AddMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) error {
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

	for _, subj := range subjects {
		vals = append(vals, fmt.Sprintf("('%s', '%s')", groupID, subj))
	}

	q := fmt.Sprintf(
		"UPSERT INTO group_members (%s, %s) VALUES %s",
		groupMemberCols.GroupID, groupMemberCols.SubjectID,
		strings.Join(vals, ", "),
	)

	_, err = tx.ExecContext(ctx, q)
	if err != nil {
		fmt.Println(err.Error())
	}

	return err
}

func (gs *groupService) ListMembers(ctx context.Context, groupID gidx.PrefixedID, pagination crdbx.Paginator) ([]gidx.PrefixedID, error) {
	paginate := crdbx.Paginate(pagination, crdbx.ContextAsOfSystemTime(ctx, nil))

	q := fmt.Sprintf(
		"SELECT %s FROM group_members %s WHERE %s = $1 %s %s %s",
		groupMemberCols.SubjectID, paginate.AsOfSystemTime(), groupMemberCols.GroupID,
		paginate.AndWhere(2), //nolint:gomnd
		paginate.OrderClause(),
		paginate.LimitClause(),
	)

	rows, err := gs.db.QueryContext(ctx, q, paginate.Values(groupID)...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

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

func (gs *groupService) RemoveMember(ctx context.Context, groupID gidx.PrefixedID, subject gidx.PrefixedID) error {
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

	_, err = tx.ExecContext(ctx, q, groupID, subject)

	return err
}

func (gs *groupService) ReplaceMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) error {
	tx, err := getContextTx(ctx)
	if err != nil {
		return err
	}

	if _, err := gs.fetchGroupByID(ctx, groupID); err != nil {
		return err
	}

	delq := fmt.Sprintf(
		"DELETE FROM group_members WHERE %s = $1",
		groupMemberCols.GroupID,
	)

	_, err = tx.ExecContext(ctx, delq, groupID)
	if err != nil {
		return err
	}

	return gs.AddMembers(ctx, groupID, subjects...)
}
