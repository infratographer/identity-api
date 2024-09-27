package events

import (
	"context"

	"go.infratographer.com/x/gidx"
)

// Service is the interface for the events service.
type Service interface {
	GroupService
}

// GroupService provides group-related event publishing and handling.
type GroupService interface {
	// AddGroupMembers adds subjects to a group.
	AddGroupMembers(ctx context.Context, gid gidx.PrefixedID, subjIDs ...gidx.PrefixedID) error
	// RemoveGroupMembers removes subjects from a group.
	RemoveGroupMembers(ctx context.Context, gid gidx.PrefixedID, subjIDs ...gidx.PrefixedID) error
	// CreateGroup creates a group.
	CreateGroup(ctx context.Context, parentID, gid gidx.PrefixedID) error
	// DeleteGroup deletes a group.
	DeleteGroup(ctx context.Context, parentID, gid gidx.PrefixedID) error
}
