package httpsrv

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go.infratographer.com/identity-api/internal/storage"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

func validationErrorHandler(ctx *gin.Context, err error, status int) {
	messages := []string{
		err.Error(),
	}

	resp := v1.ErrorResponse{
		Errors: messages,
	}

	ctx.JSON(status, resp)
}

func buildSingleErrorResponse(ctx *gin.Context) {
	err := ctx.Errors[0]

	switch e := err.Err.(type) {
	case errorWithStatus:
		resp := v1.ErrorResponse{
			Errors: []string{
				e.message,
			},
		}

		ctx.JSON(e.status, resp)
	default:
		buildMultiErrorResponse(ctx)
	}
}

func buildMultiErrorResponse(ctx *gin.Context) {
	messages := make([]string, len(ctx.Errors))
	for i, err := range ctx.Errors {
		messages[i] = err.Error()
	}

	resp := v1.ErrorResponse{
		Errors: messages,
	}

	ctx.JSON(http.StatusInternalServerError, resp)
}

func errorHandlerMiddleware(ctx *gin.Context) {
	ctx.Next()

	switch len(ctx.Errors) {
	case 0:
		return
	case 1:
		buildSingleErrorResponse(ctx)
	default:
		buildMultiErrorResponse(ctx)
	}
}

func storageMiddleware(engine storage.Engine) gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		reqCtx := gCtx.Request.Context()

		newCtx, err := engine.BeginContext(reqCtx)
		if err != nil {
			resp := v1.ErrorResponse{
				Errors: []string{
					err.Error(),
				},
			}

			gCtx.AbortWithStatusJSON(http.StatusBadGateway, resp)

			return
		}

		gCtx.Request = gCtx.Request.WithContext(newCtx)

		gCtx.Next()

		if len(gCtx.Errors) == 0 {
			err = engine.CommitContext(newCtx)
			if err != nil {
				err = errorWithStatus{
					status:  http.StatusBadGateway,
					message: err.Error(),
				}
				gCtx.Error(err) //nolint:errcheck
			}

			return
		}

		err = engine.RollbackContext(newCtx)
		if err != nil {
			err = errorWithStatus{
				status:  http.StatusBadGateway,
				message: err.Error(),
			}
			gCtx.Error(err) //nolint:errcheck
		}
	}
}

// apiHandler represents an API handler.
type apiHandler struct {
	engine storage.Engine
}

// APIHandler represents an identity-api management API handler.
type APIHandler struct {
	handler              *apiHandler
	validationMiddleware gin.HandlerFunc
}

// NewAPIHandler creates an API handler with the given storage engine.
func NewAPIHandler(engine storage.Engine) (*APIHandler, error) {
	validationMiddleware, err := oapiValidationMiddleware()
	if err != nil {
		return nil, err
	}

	handler := apiHandler{
		engine: engine,
	}

	out := &APIHandler{
		handler:              &handler,
		validationMiddleware: validationMiddleware,
	}

	return out, nil
}

// Routes registers the API's routes against the provided router group.
func (h *APIHandler) Routes(rg *gin.RouterGroup) {
	rg.Use(
		h.validationMiddleware,
		errorHandlerMiddleware,
		storageMiddleware(h.handler.engine),
	)

	options := GinServerOptions{
		ErrorHandler: validationErrorHandler,
	}

	strictHandler := NewStrictHandler(h.handler, nil)

	RegisterHandlersWithOptions(rg, strictHandler, options)
}
