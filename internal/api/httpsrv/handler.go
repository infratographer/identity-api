package httpsrv

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/metal-toolbox/auditevent/middleware/echoaudit"

	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/permissions-api/pkg/permissions"
)

func storageMiddleware(engine storage.Engine) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(eCtx echo.Context) error {
			reqCtx := eCtx.Request().Context()

			newCtx, err := engine.BeginContext(reqCtx)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadGateway, err)
			}

			eCtx.SetRequest(eCtx.Request().WithContext(newCtx))

			if err := next(eCtx); err != nil {
				eCtx.Error(err)

				rbErr := engine.RollbackContext(newCtx)
				if rbErr != nil {
					return echo.NewHTTPError(http.StatusBadGateway, rbErr, err)
				}

				return err
			}

			if err = engine.CommitContext(newCtx); err != nil {
				return echo.NewHTTPError(http.StatusBadGateway, err)
			}

			return nil
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
	validationMiddleware echo.MiddlewareFunc
	auditMiddleware      *echoaudit.Middleware
	permsMiddleware      *permissions.Permissions
}

// NewAPIHandler creates an API handler with the given storage engine.
func NewAPIHandler(engine storage.Engine, amw *echoaudit.Middleware, pmw *permissions.Permissions) (*APIHandler, error) {
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
		auditMiddleware:      amw,
		permsMiddleware:      pmw,
	}

	return out, nil
}

// Routes registers the API's routes against the provided router group.
func (h *APIHandler) Routes(rg *echo.Group) {
	middleware := []echo.MiddlewareFunc{
		h.validationMiddleware,
		storageMiddleware(h.handler.engine),
		h.permsMiddleware.Middleware(),
	}

	if h.auditMiddleware != nil {
		middleware = append(middleware, h.auditMiddleware.Audit())
	}

	rg.Use(middleware...)

	strictHandler := NewStrictHandler(h.handler, nil)

	RegisterHandlers(rg, strictHandler)
}
