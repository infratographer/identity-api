package httpsrv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/metal-toolbox/auditevent/middleware/echoaudit"

	"go.infratographer.com/identity-api/internal/events"
	"go.infratographer.com/identity-api/internal/storage"
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
	engine       storage.Engine
	eventService events.Service
}

// APIHandler represents an identity-api management API handler.
type APIHandler struct {
	handler              *apiHandler
	validationMiddleware echo.MiddlewareFunc
	auditMiddleware      *echoaudit.Middleware
	middleware           []echo.MiddlewareFunc
}

// NewAPIHandler creates an API handler with the given storage engine.
func NewAPIHandler(
	engine storage.Engine, es events.Service,
	amw *echoaudit.Middleware, middleware ...echo.MiddlewareFunc,
) (*APIHandler, error) {
	validationMiddleware, err := oapiValidationMiddleware()
	if err != nil {
		return nil, err
	}

	handler := apiHandler{
		engine:       engine,
		eventService: es,
	}

	out := &APIHandler{
		handler:              &handler,
		validationMiddleware: validationMiddleware,
		auditMiddleware:      amw,
		middleware:           middleware,
	}

	return out, nil
}

// Routes registers the API's routes against the provided router group.
func (h *APIHandler) Routes(rg *echo.Group) {
	middleware := []echo.MiddlewareFunc{
		h.validationMiddleware,
		storageMiddleware(h.handler.engine),
	}

	if h.auditMiddleware != nil {
		middleware = append(middleware, h.auditMiddleware.Audit())
	}

	middleware = append(middleware, h.middleware...)

	rg.Use(middleware...)

	strictHandler := NewStrictHandler(h.handler, nil)

	RegisterHandlers(rg, strictHandler)
}

func (h *apiHandler) rollbackAndReturnError(ctx context.Context, httpcode int, msg string) *echo.HTTPError {
	if err := h.engine.RollbackContext(ctx); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			fmt.Errorf("%s and %w", msg, ErrDBRollbackFailed),
		).SetInternal(err)
	}

	return echo.NewHTTPError(httpcode, msg)
}
