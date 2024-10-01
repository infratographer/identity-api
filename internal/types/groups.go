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
	// CreateGroup creates a new group.
	CreateGroup(ctx context.Context, group Group) (*Group, error)
	// GetGroupByID retrieves a group by its ID.
	GetGroupByID(ctx context.Context, id gidx.PrefixedID) (*Group, error)
	// UpdateGroup updates a group.
	UpdateGroup(ctx context.Context, id gidx.PrefixedID, update GroupUpdate) (*Group, error)
	// DeleteGroup deletes a group.
	DeleteGroup(ctx context.Context, id gidx.PrefixedID) error

	// ListGroupsByOwner retrieves a list of groups owned by an OU.
	ListGroupsByOwner(ctx context.Context, ownerID gidx.PrefixedID, pagination crdbx.Paginator) (Groups, error)
	// ListGroupsBySubject retrieves a list of groups that a subject is a member of.
	ListGroupsBySubject(ctx context.Context, subject gidx.PrefixedID, pagination crdbx.Paginator) (Groups, error)

	// AddGroupMembers adds subjects to a group.
	AddGroupMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) error
	// ListGroupMembers retrieves a list of subjects in a group.
	ListGroupMembers(ctx context.Context, groupID gidx.PrefixedID, pagination crdbx.Paginator) ([]gidx.PrefixedID, error)
	// RemoveGroupMember removes a subject from a group.
	RemoveGroupMember(ctx context.Context, groupID gidx.PrefixedID, subject gidx.PrefixedID) error
	// ReplaceGroupMembers replaces the members of a group with a new set of subjects.
	ReplaceGroupMembers(ctx context.Context, groupID gidx.PrefixedID, subjects ...gidx.PrefixedID) (add, rm []gidx.PrefixedID, err error)
	// GroupMembersCount retrieves the number of members in a group.
	GroupMembersCount(ctx context.Context, groupID gidx.PrefixedID) (int, error)
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

// ToPrefixedIDs converts a list of groups to a list of group IDs.
func (g Groups) ToPrefixedIDs() []gidx.PrefixedID {
	out := make([]gidx.PrefixedID, len(g))

	for i, group := range g {
		out[i] = group.ID
	}

	return out
}
