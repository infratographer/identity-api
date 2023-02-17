package httpsrv

import (
	middleware "github.com/deepmap/oapi-codegen/pkg/gin-middleware"
	"github.com/gin-gonic/gin"

	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

func oapiValidationMiddleware() (gin.HandlerFunc, error) {
	swagger, err := v1.GetSwagger()
	if err != nil {
		return nil, err
	}

	return middleware.OapiRequestValidator(swagger), nil
}
