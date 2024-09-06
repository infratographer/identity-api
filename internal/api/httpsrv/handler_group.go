package httpsrv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/identity-api/internal/types"
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
