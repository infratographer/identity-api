package events

import (
	"context"

	"go.infratographer.com/permissions-api/pkg/permissions"
	eventsx "go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"
)

const (
	// DirectMemberRelationship is the direct member relationship.
	DirectMemberRelationship = "direct_member"
	// GroupParentRelationship is the group parent relationship.
	GroupParentRelationship = "parent"
	// GroupTopic is the group topic.
	GroupTopic = "group"
)

// AddGroupMembers adds subjects to a group.
func (e *Events) AddGroupMembers(ctx context.Context, gid gidx.PrefixedID, subjIDs ...gidx.PrefixedID) error {
	if len(subjIDs) == 0 {
		return nil
	}

	rels := make([]eventsx.AuthRelationshipRelation, 0, len(subjIDs))

	for _, subj := range subjIDs {
		if subj == "" {
			continue
		}

		rels = append(rels,
			eventsx.AuthRelationshipRelation{
				Relation:  DirectMemberRelationship,
				SubjectID: subj,
			},
		)
	}

	if len(rels) == 0 {
		return nil
	}

	return permissions.CreateAuthRelationships(ctx, GroupTopic, gid, rels...)
}

// RemoveGroupMembers removes subjects from a group.
func (e *Events) RemoveGroupMembers(ctx context.Context, gid gidx.PrefixedID, subjIDs ...gidx.PrefixedID) error {
	rels := make([]eventsx.AuthRelationshipRelation, 0, len(subjIDs))

	for _, subj := range subjIDs {
		if subj == "" {
			continue
		}

		rels = append(rels,
			eventsx.AuthRelationshipRelation{
				Relation:  DirectMemberRelationship,
				SubjectID: subj,
			},
		)
	}

	if len(rels) == 0 {
		return nil
	}

	return permissions.DeleteAuthRelationships(ctx, GroupTopic, gid, rels...)
}

// CreateGroup creates a group.
func (e *Events) CreateGroup(ctx context.Context, parentID, gid gidx.PrefixedID) error {
	return permissions.CreateAuthRelationships(
		ctx, GroupTopic, gid,
		eventsx.AuthRelationshipRelation{
			Relation:  GroupParentRelationship,
			SubjectID: parentID,
		},
	)
}

// DeleteGroup deletes a group.
func (e *Events) DeleteGroup(ctx context.Context, parentID, gid gidx.PrefixedID) error {
	return permissions.DeleteAuthRelationships(
		ctx, GroupTopic, gid,
		eventsx.AuthRelationshipRelation{
			Relation:  GroupParentRelationship,
			SubjectID: parentID,
		},
	)
}
