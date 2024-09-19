package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/permissions-api/pkg/permissions"
	"go.infratographer.com/x/gidx"
)

const (
	actionGroupMembersList   = "iam_group_members_list"
	actionGroupMembersAdd    = "iam_group_members_add"
	actionGroupMembersPut    = "iam_group_members_put"
	actionGroupMembersRemove = "iam_group_members_remove"
)

// AddGroupMembers creates a group
func (h *apiHandler) AddGroupMembers(ctx context.Context, req AddGroupMembersRequestObject) (AddGroupMembersResponseObject, error) {
	reqbody := req.Body
	gid := req.GroupID

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid owner id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupMembersAdd); err != nil {
		return nil, permissionsError(err)
	}

	if err := h.engine.AddMembers(ctx, gid, reqbody.MemberIDs...); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	return AddGroupMembers200JSONResponse{Ok: true}, nil
}

// ListGroupMembers lists the members of a group
func (h *apiHandler) ListGroupMembers(ctx context.Context, req ListGroupMembersRequestObject) (ListGroupMembersResponseObject, error) {
	gid := req.GroupID

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupMembersList); err != nil {
		return nil, permissionsError(err)
	}

	members, err := h.engine.ListMembers(ctx, gid, req.Params)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	collection := v1.GroupMemberCollection{
		Members:    members,
		GroupID:    gid,
		Pagination: v1.Pagination{},
	}

	if err := req.Params.SetPagination(&collection); err != nil {
		return nil, err
	}

	return ListGroupMembers200JSONResponse{GroupMemberCollectionJSONResponse(collection)}, nil
}

// RemoveGroupMember removes a member from a group
func (h *apiHandler) RemoveGroupMember(ctx context.Context, req RemoveGroupMemberRequestObject) (RemoveGroupMemberResponseObject, error) {
	gid := req.GroupID
	sid := req.SubjectID

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	if _, err := gidx.Parse(string(sid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid member id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupMembersRemove); err != nil {
		return nil, permissionsError(err)
	}

	if err := h.engine.RemoveMember(ctx, gid, sid); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	return RemoveGroupMember200JSONResponse{true}, nil
}

// ReplaceGroupMembers replaces the members of a group
func (h *apiHandler) ReplaceGroupMembers(ctx context.Context, req ReplaceGroupMembersRequestObject) (ReplaceGroupMembersResponseObject, error) {
	gid := req.GroupID
	reqbody := req.Body

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupMembersPut); err != nil {
		return nil, permissionsError(err)
	}

	if err := h.engine.ReplaceMembers(ctx, gid, reqbody.MemberIDs...); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	return ReplaceGroupMembers200JSONResponse{true}, nil
}
