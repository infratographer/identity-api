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
	actionGroupGet    = "iam_group_get"
	actionGroupList   = "iam_group_list"
	actionGroupCreate = "iam_group_create"
	actionGroupUpdate = "iam_group_update"
	actionGroupDelete = "iam_group_delete"
)

// CreateGroup creates a group
func (h *apiHandler) CreateGroup(ctx context.Context, req CreateGroupRequestObject) (CreateGroupResponseObject, error) {
	reqbody := req.Body
	ownerID := req.OwnerID

	if _, err := gidx.Parse(string(ownerID)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid owner id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, ownerID, actionGroupCreate); err != nil {
		return nil, permissionsError(err)
	}

	id, err := gidx.NewID(types.IdentityGroupIDPrefix)
	if err != nil {
		err = echo.NewHTTPError(
			http.StatusInternalServerError,
			fmt.Sprintf("failed to generate new id: %s", err.Error()),
		)

		return nil, err
	}

	description := ""
	if reqbody.Description != nil {
		description = *reqbody.Description
	}

	g, err := h.engine.CreateGroup(ctx, types.Group{
		ID:          id,
		OwnerID:     ownerID,
		Name:        reqbody.Name,
		Description: description,
	})
	if err != nil {
		if errors.Is(err, types.ErrGroupExists) {
			err = echo.NewHTTPError(
				http.StatusConflict,
				fmt.Sprintf("group \"%s\" already exists", reqbody.Name),
			)
		} else if errors.Is(err, types.ErrInvalidArgument) {
			err = echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return nil, err
	}

	if err := h.eventService.CreateGroup(ctx, ownerID, id); err != nil {
		if err := h.engine.RollbackContext(ctx); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadGateway, err)
		}

		return nil, echo.NewHTTPError(
			http.StatusBadGateway,
			"failed to create group in permissions API",
		)
	}

	groupResp, err := g.ToV1Group()
	if err != nil {
		return nil, err
	}

	return CreateGroup200JSONResponse(groupResp), nil
}

// GetGroupByID fetches a group from storage by its ID
func (h *apiHandler) GetGroupByID(ctx context.Context, req GetGroupByIDRequestObject) (GetGroupByIDResponseObject, error) {
	gid := req.GroupID

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupGet); err != nil {
		return nil, permissionsError(err)
	}

	g, err := h.engine.GetGroupByID(ctx, gid)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			err = echo.NewHTTPError(
				http.StatusNotFound,
				fmt.Sprintf("group %s not found", gid),
			)

			return nil, err
		}

		return nil, err
	}

	groupResp, err := g.ToV1Group()
	if err != nil {
		return nil, err
	}

	return GetGroupByID200JSONResponse(groupResp), nil
}

// ListGroups fetches a list of groups from storage
func (h *apiHandler) ListGroups(ctx context.Context, req ListGroupsRequestObject) (ListGroupsResponseObject, error) {
	ownerID := req.OwnerID

	if _, err := gidx.Parse(string(ownerID)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid owner id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, ownerID, actionGroupList); err != nil {
		return nil, permissionsError(err)
	}

	groups, err := h.engine.ListGroupsByOwner(ctx, ownerID, req.Params)
	if err != nil {
		return nil, err
	}

	groupResp, err := groups.ToV1Groups()
	if err != nil {
		return nil, err
	}

	collection := v1.GroupCollection{
		Groups:     groupResp,
		Pagination: v1.Pagination{},
	}

	if err := req.Params.SetPagination(&collection); err != nil {
		return nil, err
	}

	return ListGroups200JSONResponse{GroupCollectionJSONResponse(collection)}, nil
}

// UpdateGroup updates a group in storage
func (h *apiHandler) UpdateGroup(ctx context.Context, req UpdateGroupRequestObject) (UpdateGroupResponseObject, error) {
	gid := req.GroupID
	reqbody := req.Body

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupUpdate); err != nil {
		return nil, permissionsError(err)
	}

	updates := types.GroupUpdate{
		Name:        reqbody.Name,
		Description: reqbody.Description,
	}

	g, err := h.engine.UpdateGroup(ctx, gid, updates)
	if err != nil {
		if errors.Is(err, types.ErrGroupNotFound) {
			err = echo.NewHTTPError(
				http.StatusNotFound,
				fmt.Sprintf("group %s not found", gid),
			)

			return nil, err
		}

		if errors.Is(err, types.ErrGroupExists) {
			err = echo.NewHTTPError(
				http.StatusConflict,
				"group with same name already exists",
			)

			return nil, err
		}

		return nil, err
	}

	groupResp, err := g.ToV1Group()
	if err != nil {
		return nil, err
	}

	return UpdateGroup200JSONResponse(groupResp), nil
}

// DeleteGroup deletes a group
func (h *apiHandler) DeleteGroup(ctx context.Context, req DeleteGroupRequestObject) (DeleteGroupResponseObject, error) {
	gid := req.GroupID

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	if err := permissions.CheckAccess(ctx, gid, actionGroupDelete); err != nil {
		return nil, permissionsError(err)
	}

	group, err := h.engine.GetGroupByID(ctx, gid)
	if err != nil {
		return nil, err
	}

	mc, err := h.engine.GroupMembersCount(ctx, gid)
	if err != nil {
		return nil, err
	}

	if mc > 0 {
		err := echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("cannot delete group %s: still has members", gid),
		)

		return nil, err
	}

	err = h.engine.DeleteGroup(ctx, group.ID)
	if err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return nil, err
	}

	if err := h.eventService.DeleteGroup(ctx, group.OwnerID, group.ID); err != nil {
		if err := h.engine.RollbackContext(ctx); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadGateway, err)
		}

		return nil, echo.NewHTTPError(
			http.StatusBadGateway,
			"failed to remove group in permissions API",
		)
	}

	return DeleteGroup200JSONResponse{true}, nil
}
