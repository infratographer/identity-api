// Package httpsrv provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.12.5-0.20230118012357-f4cf8f9a5703 DO NOT EDIT.
package httpsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/gin-gonic/gin"
	. "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Creates an issuer.
	// (POST /api/v1/issuers)
	CreateIssuer(c *gin.Context)
	// Gets an issuer by ID.
	// (GET /api/v1/issuers/{id})
	GetIssuerByID(c *gin.Context, id openapi_types.UUID)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandler       func(*gin.Context, error, int)
}

type MiddlewareFunc func(c *gin.Context)

// CreateIssuer operation middleware
func (siw *ServerInterfaceWrapper) CreateIssuer(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.CreateIssuer(c)
}

// GetIssuerByID operation middleware
func (siw *ServerInterfaceWrapper) GetIssuerByID(c *gin.Context) {

	var err error

	// ------------- Path parameter "id" -------------
	var id openapi_types.UUID

	err = runtime.BindStyledParameter("simple", false, "id", c.Param("id"), &id)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter id: %s", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetIssuerByID(c, id)
}

// GinServerOptions provides options for the Gin server.
type GinServerOptions struct {
	BaseURL      string
	Middlewares  []MiddlewareFunc
	ErrorHandler func(*gin.Context, error, int)
}

// RegisterHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterHandlers(router gin.IRouter, si ServerInterface) {
	RegisterHandlersWithOptions(router, si, GinServerOptions{})
}

// RegisterHandlersWithOptions creates http.Handler with additional options
func RegisterHandlersWithOptions(router gin.IRouter, si ServerInterface, options GinServerOptions) {
	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"msg": err.Error()})
		}
	}

	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandler:       errorHandler,
	}

	router.POST(options.BaseURL+"/api/v1/issuers", wrapper.CreateIssuer)
	router.GET(options.BaseURL+"/api/v1/issuers/:id", wrapper.GetIssuerByID)
}

type CreateIssuerRequestObject struct {
	Body *CreateIssuerJSONRequestBody
}

type CreateIssuerResponseObject interface {
	VisitCreateIssuerResponse(w http.ResponseWriter) error
}

type CreateIssuer200JSONResponse Issuer

func (response CreateIssuer200JSONResponse) VisitCreateIssuerResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type CreateIssuer400JSONResponse ErrorResponse

func (response CreateIssuer400JSONResponse) VisitCreateIssuerResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type GetIssuerByIDRequestObject struct {
	Id openapi_types.UUID `json:"id"`
}

type GetIssuerByIDResponseObject interface {
	VisitGetIssuerByIDResponse(w http.ResponseWriter) error
}

type GetIssuerByID200JSONResponse Issuer

func (response GetIssuerByID200JSONResponse) VisitGetIssuerByIDResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetIssuerByID404JSONResponse ErrorResponse

func (response GetIssuerByID404JSONResponse) VisitGetIssuerByIDResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(404)

	return json.NewEncoder(w).Encode(response)
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// Creates an issuer.
	// (POST /api/v1/issuers)
	CreateIssuer(ctx context.Context, request CreateIssuerRequestObject) (CreateIssuerResponseObject, error)
	// Gets an issuer by ID.
	// (GET /api/v1/issuers/{id})
	GetIssuerByID(ctx context.Context, request GetIssuerByIDRequestObject) (GetIssuerByIDResponseObject, error)
}

type StrictHandlerFunc func(ctx *gin.Context, args interface{}) (interface{}, error)

type StrictMiddlewareFunc func(f StrictHandlerFunc, operationID string) StrictHandlerFunc

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
}

// CreateIssuer operation middleware
func (sh *strictHandler) CreateIssuer(ctx *gin.Context) {
	var request CreateIssuerRequestObject

	var body CreateIssuerJSONRequestBody
	if err := ctx.ShouldBind(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		ctx.Error(err)
		return
	}
	request.Body = &body

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.CreateIssuer(ctx, request.(CreateIssuerRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "CreateIssuer")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
	} else if validResponse, ok := response.(CreateIssuerResponseObject); ok {
		if err := validResponse.VisitCreateIssuerResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("Unexpected response type: %T", response))
	}
}

// GetIssuerByID operation middleware
func (sh *strictHandler) GetIssuerByID(ctx *gin.Context, id openapi_types.UUID) {
	var request GetIssuerByIDRequestObject

	request.Id = id

	handler := func(ctx *gin.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetIssuerByID(ctx, request.(GetIssuerByIDRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetIssuerByID")
	}

	response, err := handler(ctx, request)

	if err != nil {
		ctx.Error(err)
	} else if validResponse, ok := response.(GetIssuerByIDResponseObject); ok {
		if err := validResponse.VisitGetIssuerByIDResponse(ctx.Writer); err != nil {
			ctx.Error(err)
		}
	} else if response != nil {
		ctx.Error(fmt.Errorf("Unexpected response type: %T", response))
	}
}
