package httpsrv

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go.infratographer.com/identity-manager-sts/internal/types"
	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

var (
	responseNotFound = v1.ErrorResponse{
		Errors: []string{
			"not found",
		},
	}
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

func errorHandlerMiddleware(ctx *gin.Context) {
	ctx.Next()

	if len(ctx.Errors) == 0 {
		return
	}

	messages := make([]string, len(ctx.Errors))
	for i, err := range ctx.Errors {
		messages[i] = err.Error()
	}

	resp := v1.ErrorResponse{
		Errors: messages,
	}

	ctx.JSON(http.StatusInternalServerError, resp)
}

// apiHandler represents an API handler.
type apiHandler struct {
	engine types.IssuerService
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
			return CreateIssuer400JSONResponse{
				Errors: []string{
					"error parsing CEL expression",
				},
			}, nil
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
		return GetIssuerByID404JSONResponse(responseNotFound), nil
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
			return UpdateIssuer400JSONResponse{
				Errors: []string{
					"error parsing CEL expression",
				},
			}, nil
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
		return UpdateIssuer404JSONResponse(responseNotFound), nil
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
	case nil:
	case types.ErrorIssuerNotFound:
		return DeleteIssuer404JSONResponse(responseNotFound), nil
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
func NewAPIHandler(engine types.IssuerService) (*APIHandler, error) {
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
	)

	options := GinServerOptions{
		ErrorHandler: validationErrorHandler,
	}

	strictHandler := NewStrictHandler(h.handler, nil)

	RegisterHandlersWithOptions(rg, strictHandler, options)
}
