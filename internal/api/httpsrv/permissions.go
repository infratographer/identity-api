package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.infratographer.com/permissions-api/pkg/permissions"
	"go.infratographer.com/x/gidx"
)

func checkAccessWithResponse(ctx context.Context, resourceID gidx.PrefixedID, action string) error {
	accessRequest := permissions.AccessRequest{
		Action:     action,
		ResourceID: resourceID,
	}

	err := permissions.CheckAll(ctx, accessRequest)

	switch {
	case errors.Is(err, permissions.ErrPermissionDenied):
		msg := fmt.Sprintf(
			"subject does not have permission to perform action '%s' on resource '%s'",
			action,
			resourceID,
		)

		return errorWithStatus{
			status:  http.StatusForbidden,
			message: msg,
		}
	case err != nil:
		return errorWithStatus{
			status:  http.StatusInternalServerError,
			message: err.Error(),
		}
	default:
		return nil
	}
}
