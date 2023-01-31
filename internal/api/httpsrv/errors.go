package httpsrv

import "net/http"

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
)
