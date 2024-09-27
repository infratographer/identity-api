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

	for _, mid := range reqbody.MemberIDs {
		if _, err := gidx.Parse(string(mid)); err != nil {
			err = echo.NewHTTPError(
				http.StatusBadRequest,
				fmt.Sprintf("invalid member id %s: %s", mid, err.Error()),
			)

			return nil, err
		}
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupMembersAdd); err != nil {
		return nil, permissionsError(err)
	}

	if err := h.engine.AddGroupMembers(ctx, gid, reqbody.MemberIDs...); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	if err := h.eventService.AddGroupMembers(ctx, gid, reqbody.MemberIDs...); err != nil {
		resperr := h.rollbackAndReturnError(ctx, http.StatusBadGateway, "failed to add group members in permissions API")
		return nil, resperr
	}

	return AddGroupMembers200JSONResponse{Success: true}, nil
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

	members, err := h.engine.ListGroupMembers(ctx, gid, req.Params)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	collection := v1.GroupMemberCollection{
		MemberIDs:  members,
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

	if err := h.engine.RemoveGroupMember(ctx, gid, sid); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	if err := h.eventService.RemoveGroupMembers(ctx, gid, sid); err != nil {
		resperr := h.rollbackAndReturnError(ctx, http.StatusBadGateway, "failed to remove group member in permissions API")
		return nil, resperr
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

	for _, mid := range reqbody.MemberIDs {
		if _, err := gidx.Parse(string(mid)); err != nil {
			err = echo.NewHTTPError(
				http.StatusBadRequest,
				fmt.Sprintf("invalid member id %s: %s", mid, err.Error()),
			)

			return nil, err
		}
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupMembersPut); err != nil {
		return nil, permissionsError(err)
	}

	current, err := h.engine.ListGroupMembers(ctx, gid, nil)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	if err := h.engine.ReplaceGroupMembers(ctx, gid, reqbody.MemberIDs...); err != nil {
		return nil, err
	}

	if err := h.eventService.RemoveGroupMembers(ctx, gid, current...); err != nil {
		resperr := h.rollbackAndReturnError(ctx, http.StatusBadGateway, "failed to replace group members in permissions API")
		return nil, resperr
	}

	if err := h.eventService.AddGroupMembers(ctx, gid, reqbody.MemberIDs...); err != nil {
		resperr := h.rollbackAndReturnError(ctx, http.StatusBadGateway, "failed to replace group members in permissions API")
		return nil, resperr
	}

	return ReplaceGroupMembers200JSONResponse{true}, nil
}

func (h *apiHandler) ListUserGroups(ctx context.Context, req ListUserGroupsRequestObject) (ListUserGroupsResponseObject, error) {
	subject := req.UserID

	if _, err := gidx.Parse(string(subject)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid subject id: %s", err.Error()),
		)

		return nil, err
	}

	// Find the owner the user's issuer is on to check permissions.
	ownerID, err := h.engine.LookupUserOwnerID(ctx, subject)
	switch err {
	case nil:
	case types.ErrUserInfoNotFound:
		return nil, echo.NewHTTPError(http.StatusNotFound, err.Error())
	default:
		return nil, err
	}

	if err := permissions.CheckAccess(ctx, ownerID, actionUserGet); err != nil {
		return nil, permissionsError(err)
	}

	groups, err := h.engine.ListGroupsBySubject(ctx, subject, req.Params)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(http.StatusNotFound, err.Error())
		}

		return nil, err
	}

	resp := groups.ToPrefixedIDs()

	collection := v1.GroupIDCollection{
		GroupIDs:   resp,
		Pagination: v1.Pagination{},
	}

	if err := req.Params.SetPagination(&collection); err != nil {
		return nil, err
	}

	return ListUserGroups200JSONResponse{GroupIDCollectionJSONResponse(collection)}, nil
}
