package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/gidx"
)

// CreateGroup creates a group
func (h *apiHandler) CreateGroup(ctx context.Context, req CreateGroupRequestObject) (CreateGroupResponseObject, error) {
	reqbody := req.Body
	ownerID := req.OwnerID

	// if err := permissions.CheckAccess(ctx, ownerID, actionGroupCreate); err != nil {
	// 	return nil, permissionsError(err)
	// }

	if _, err := gidx.Parse(string(ownerID)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid owner id: %s", err.Error()),
		)

		return nil, err
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
		return nil, err
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

	// if err := permissions.CheckAccess(ctx, gid, actionGroupRead); err != nil {
	// 	return nil, permissionsError(err)
	// }

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
	}

	g, err := h.engine.GetGroupByID(ctx, gid)
	if err != nil {
		if errors.Is(err, types.ErrGroupNotFound) {
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

	// if err := permissions.CheckAccess(ctx, ownerID, actionGroupList); err != nil {
	// 	return nil, permissionsError(err)
	// }

	if _, err := gidx.Parse(string(ownerID)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid owner id: %s", err.Error()),
		)

		return nil, err
	}

	groups, err := h.engine.ListGroups(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	var groupResp []v1.Group

	for _, g := range groups {
		g, err := g.ToV1Group()
		if err != nil {
			return nil, err
		}

		groupResp = append(groupResp, g)
	}

	return ListGroups200JSONResponse(groupResp), nil
}

// UpdateGroup updates a group in storage
func (h *apiHandler) UpdateGroup(ctx context.Context, req UpdateGroupRequestObject) (UpdateGroupResponseObject, error) {
	gid := req.GroupID
	reqbody := req.Body

	// if err := permissions.CheckAccess(ctx, gid, actionGroupUpdate); err != nil {
	// 	return nil, permissionsError(err)
	// }

	if _, err := gidx.Parse(string(gid)); err != nil {
		err = echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("invalid group id: %s", err.Error()),
		)

		return nil, err
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

		return nil, err
	}

	groupResp, err := g.ToV1Group()
	if err != nil {
		return nil, err
	}

	return UpdateGroup200JSONResponse(groupResp), nil
}
