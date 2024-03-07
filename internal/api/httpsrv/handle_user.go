package httpsrv

import (
	"context"
	"net/http"

	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/permissions-api/pkg/permissions"
)

const (
	actionUserGet = "iam_user_get"
)

func (h *apiHandler) GetUserByID(ctx context.Context, req GetUserByIDRequestObject) (GetUserByIDResponseObject, error) {
	// Find the owner the user's issuer is on to check permissions.
	ownerID, err := h.engine.LookupUserOwnerID(ctx, req.UserID)
	switch err {
	case nil:
	case types.ErrUserInfoNotFound:
		return nil, errorWithStatus{
			status:  http.StatusNotFound,
			message: err.Error(),
		}
	default:
		return nil, err
	}

	if err := permissions.CheckAccess(ctx, ownerID, actionUserGet); err != nil {
		return nil, permissionsError(err)
	}

	info, err := h.engine.LookupUserInfoByID(ctx, req.UserID)
	switch err {
	case nil:
	case types.ErrUserInfoNotFound:
		return nil, errorWithStatus{
			status:  http.StatusNotFound,
			message: err.Error(),
		}
	default:
		return nil, err
	}

	out, err := info.ToV1User()
	if err != nil {
		return nil, err
	}

	return GetUserByID200JSONResponse(out), nil
}
