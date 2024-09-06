package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
