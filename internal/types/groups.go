package types

import (
	"context"

	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/gidx"
)

// Group represents a set of subjects
type Group struct {
	// ID is the group's ID
	ID gidx.PrefixedID
	// OwnerID is the ID of the OU that owns the group
	OwnerID gidx.PrefixedID
	// Name is the group's name
	Name string
	// Description is the group's description
	Description string
}

// ToV1Group converts a group to an API group.
func (g *Group) ToV1Group() (v1.Group, error) {
	group := v1.Group{
		ID:    g.ID,
		Owner: g.OwnerID,
		Name:  g.Name,
	}

	if g.Description != "" {
		group.Description = &g.Description
	}

	return group, nil
}

// GroupUpdate represents an update operation on a group.
type GroupUpdate struct {
	Name        *string
	Description *string
}

// GroupService represents a service for managing groups.
type GroupService interface {
	CreateGroup(ctx context.Context, group Group) (*Group, error)
	GetGroupByID(ctx context.Context, id gidx.PrefixedID) (*Group, error)
	ListGroups(ctx context.Context, ownerID gidx.PrefixedID) ([]*Group, error)
	// UpdateGroup(ctx context.Context, id gidx.PrefixedID, update GroupUpdate) (*Group, error)
	// DeleteGroup(ctx context.Context, id gidx.PrefixedID) error
}
