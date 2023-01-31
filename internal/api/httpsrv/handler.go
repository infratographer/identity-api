package httpsrv

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go.infratographer.com/identity-manager-sts/internal/storage"
	"go.infratographer.com/identity-manager-sts/internal/types"
	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
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

func (h *apiHandler) CreateIssuer(ctx context.Context, req CreateIssuerRequestObject) (CreateIssuerResponseObject, error) {
	tenantID := req.TenantID
	createOp := req.Body

	var (
		claimsMapping types.ClaimsMapping
		err           error
	)

	if createOp.ClaimMappings != nil {
		claimsMapping, err = types.NewClaimsMapping(*createOp.ClaimMappings)
		if err != nil {
			err = errorWithStatus{
				status:  http.StatusBadRequest,
				message: "error parsing CEL expression",
			}

			return nil, err
		}
	}

	issuerToCreate := types.Issuer{
		TenantID:      tenantID.String(),
		ID:            uuid.New().String(),
		Name:          createOp.Name,
		URI:           createOp.URI,
		JWKSURI:       createOp.JWKSURI,
		ClaimMappings: claimsMapping,
	}

	issuer, err := h.engine.CreateIssuer(ctx, issuerToCreate)
	if err != nil {
		return nil, err
	}

	out, err := issuer.ToV1Issuer()
	if err != nil {
		return nil, err
	}

	return CreateIssuer200JSONResponse(out), nil
}

func (h *apiHandler) GetIssuerByID(ctx context.Context, req GetIssuerByIDRequestObject) (GetIssuerByIDResponseObject, error) {
	id := req.Id.String()

	iss, err := h.engine.GetIssuerByID(ctx, id)
	switch err {
	case nil:
	case types.ErrorIssuerNotFound:
		return nil, errorNotFound
	default:
		return nil, err
	}

	out, err := iss.ToV1Issuer()
	if err != nil {
		return nil, err
	}

	return GetIssuerByID200JSONResponse(out), nil
}

func (h *apiHandler) UpdateIssuer(ctx context.Context, req UpdateIssuerRequestObject) (UpdateIssuerResponseObject, error) {
	id := req.Id.String()
	updateOp := req.Body

	var (
		claimsMapping types.ClaimsMapping
		err           error
	)

	if updateOp.ClaimMappings != nil {
		claimsMapping, err = types.NewClaimsMapping(*updateOp.ClaimMappings)
		if err != nil {
			err = errorWithStatus{
				status:  http.StatusBadRequest,
				message: "error parsing CEL expression",
			}

			return nil, err
		}
	}

	update := types.IssuerUpdate{
		Name:          updateOp.Name,
		URI:           updateOp.URI,
		JWKSURI:       updateOp.JWKSURI,
		ClaimMappings: claimsMapping,
	}

	issuer, err := h.engine.UpdateIssuer(ctx, id, update)
	switch err {
	case nil:
	case types.ErrorIssuerNotFound:
		return nil, errorNotFound
	default:
		return nil, err
	}

	out, err := issuer.ToV1Issuer()
	if err != nil {
		return nil, err
	}

	return UpdateIssuer200JSONResponse(out), nil
}

func (h *apiHandler) DeleteIssuer(ctx context.Context, req DeleteIssuerRequestObject) (DeleteIssuerResponseObject, error) {
	id := req.Id.String()

	err := h.engine.DeleteIssuer(ctx, id)
	switch err {
	case nil, types.ErrorIssuerNotFound:
	default:
		return nil, err
	}

	out := v1.DeleteResponse{
		Success: true,
	}

	return DeleteIssuer200JSONResponse(out), nil
}

// APIHandler represents an identity-manager-sts management API handler.
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
