package types

import (
	"context"

	"go.infratographer.com/identity-api/internal/crdbx"
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
		ID:      g.ID,
		Name:    g.Name,
		OwnerID: &g.OwnerID,
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
	UpdateGroup(ctx context.Context, id gidx.PrefixedID, update GroupUpdate) (*Group, error)
	DeleteGroup(ctx context.Context, id gidx.PrefixedID) error

	ListGroupsByOwner(ctx context.Context, ownerID gidx.PrefixedID, pagination crdbx.Paginator) (Groups, error)
	ListGroupsBySubject(ctx context.Context, subject gidx.PrefixedID, pagination crdbx.Paginator) (Groups, error)

	AddMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) error
	ListMembers(ctx context.Context, groupID gidx.PrefixedID, pagination crdbx.Paginator) ([]gidx.PrefixedID, error)
	RemoveMember(ctx context.Context, groupID gidx.PrefixedID, subject gidx.PrefixedID) error
	ReplaceMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) error
}

// Groups represents a list of groups
type Groups []*Group

// ToV1Groups converts a list of groups to a list of API groups.
func (g Groups) ToV1Groups() ([]v1.Group, error) {
	out := make([]v1.Group, len(g))

	for i, group := range g {
		v1Group, err := group.ToV1Group()
		if err != nil {
			return nil, err
		}

		out[i] = v1Group
	}

	return out, nil
}
