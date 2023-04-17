package httpsrv

import (
	"github.com/labstack/echo/v4"

	"github.com/deepmap/oapi-codegen/pkg/middleware"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

func oapiValidationMiddleware() (echo.MiddlewareFunc, error) {
	swagger, err := v1.GetSwagger()
	if err != nil {
		return nil, err
	}

	return middleware.OapiRequestValidator(swagger), nil
}
