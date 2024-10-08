package httpsrv

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/permissions-api/pkg/permissions"
)

type errorWithStatus struct {
	status  int
	message string
}

func (e errorWithStatus) Error() string {
	return e.message
}

var (
	errorNotFound = errorWithStatus{
		status:  http.StatusNotFound,
		message: "not found",
	}

	// ErrDBRollbackFailed is returned when a database rollback fails
	ErrDBRollbackFailed = errors.New("failed to rollback database transaction")
)

func permissionsError(err error) error {
	if errors.Is(err, permissions.ErrPermissionDenied) {
		return echo.NewHTTPError(http.StatusForbidden, err)
	}

	return err
}
