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

	claimsMappingRepr, err := issuer.ClaimMappings.Repr()
	if err != nil {
		return nil, err
	}

	out := v1.Issuer{
		ID:            uuid.MustParse(issuer.ID),
		Name:          issuer.Name,
		URI:           issuer.URI,
		JWKSURI:       issuer.JWKSURI,
		ClaimMappings: claimsMappingRepr,
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

	claimMappingRepr, err := iss.ClaimMappings.Repr()
	if err != nil {
		return nil, err
	}

	out := v1.Issuer{
		ID:            uuid.MustParse(iss.ID),
		Name:          iss.Name,
		URI:           iss.URI,
		JWKSURI:       iss.JWKSURI,
		ClaimMappings: claimMappingRepr,
	}

	return GetIssuerByID200JSONResponse(out), nil
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
