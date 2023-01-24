package httpsrv

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go.infratographer.com/identity-manager-sts/internal/storage"
	"go.infratographer.com/identity-manager-sts/internal/types"
	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

// apiHandler represents an API handler.
type apiHandler struct {
	engine storage.Engine
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

	issuer, err := h.engine.Create(ctx, issuerToCreate)
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

	iss, err := h.engine.GetByID(ctx, id)
	if err != nil {
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
	strictHandler := NewStrictHandler(h.handler, nil)

	options := GinServerOptions{
		Middlewares: []MiddlewareFunc{
			MiddlewareFunc(h.validationMiddleware),
		},
	}

	RegisterHandlersWithOptions(rg, strictHandler, options)
}
