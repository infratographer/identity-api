package httpsrv

import (
	"context"
	"net/http"

	"go.infratographer.com/identity-api/internal/crypto"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/permissions-api/pkg/permissions"
)

const defaultTokenLength = 26

const (
	actionOAuthClientCreate = "iam_oauthclient_create"
	actionOAuthClientDelete = "iam_oauthclient_delete"
	actionOAuthClientGet    = "iam_oauthclient_get"
	actionOAuthClientList   = "iam_oauthclient_list"
)

// CreateOAuthClient creates a client for a owner with a set name.
// This endpoint returns the OAuth client ID and secret that the client
// needs to provide to authenticate when requesting a token.
func (h *apiHandler) CreateOAuthClient(ctx context.Context, request CreateOAuthClientRequestObject) (CreateOAuthClientResponseObject, error) {
	var newClient types.OAuthClient
	newClient.OwnerID = request.OwnerID
	newClient.Name = request.Body.Name

	if err := permissions.CheckAccess(ctx, newClient.OwnerID, actionOAuthClientCreate); err != nil {
		return nil, permissionsError(err)
	}

	newClient.Audience = []string{}
	if request.Body.Audience != nil {
		newClient.Audience = *request.Body.Audience
	}

	secret, err := crypto.GenerateSecureToken(defaultTokenLength)
	if err != nil {
		return nil, err
	}

	generatedSecret := string(secret)
	newClient.Secret = generatedSecret

	newClient, err = h.engine.CreateOAuthClient(ctx, newClient)
	if err != nil {
		return nil, err
	}

	resp := newClient.ToV1OAuthClient()

	// the object now contains the hashed secret, but the response should contain the raw secret
	resp.Secret = &generatedSecret

	return CreateOAuthClient200JSONResponse(resp), nil
}

// GetOAuthClient returns the OAuth client for that ID
func (h *apiHandler) GetOAuthClient(ctx context.Context, request GetOAuthClientRequestObject) (GetOAuthClientResponseObject, error) {
	client, err := h.engine.LookupOAuthClientByID(ctx, request.ClientID)
	switch err {
	case nil:
	case types.ErrOAuthClientNotFound:
		return nil, errorWithStatus{
			status:  http.StatusNotFound,
			message: err.Error(),
		}
	default:
		return nil, err
	}

	if err := permissions.CheckAccess(ctx, client.OwnerID, actionOAuthClientGet); err != nil {
		return nil, permissionsError(err)
	}

	apiType := client.ToV1OAuthClient()

	return GetOAuthClient200JSONResponse(apiType), nil
}

func (h *apiHandler) GetOwnerOAuthClients(ctx context.Context, req GetOwnerOAuthClientsRequestObject) (GetOwnerOAuthClientsResponseObject, error) {
	if err := permissions.CheckAccess(ctx, req.OwnerID, actionOAuthClientList); err != nil {
		return nil, permissionsError(err)
	}

	iss, err := h.engine.GetOwnerOAuthClients(ctx, req.OwnerID, req.Params)
	if err != nil {
		return nil, err
	}

	clients, err := iss.ToV1OAuthClients()
	if err != nil {
		return nil, err
	}

	collection := v1.OAuthClientCollection{
		Clients:    clients,
		Pagination: v1.Pagination{},
	}

	if err := req.Params.SetPagination(&collection); err != nil {
		return nil, err
	}

	out := OAuthClientCollectionJSONResponse(collection)

	return GetOwnerOAuthClients200JSONResponse{out}, nil
}

// DeleteOAuthClient removes the OAuth client.
func (h *apiHandler) DeleteOAuthClient(ctx context.Context, request DeleteOAuthClientRequestObject) (DeleteOAuthClientResponseObject, error) {
	// We must fetch the oauth client to retrieve the owner so we may check for permission to delete.
	client, err := h.engine.LookupOAuthClientByID(ctx, request.ClientID)
	switch err {
	case nil:
	case types.ErrOAuthClientNotFound:
		return nil, errorWithStatus{
			status:  http.StatusNotFound,
			message: err.Error(),
		}
	default:
		return nil, err
	}

	if err := permissions.CheckAccess(ctx, client.OwnerID, actionOAuthClientDelete); err != nil {
		return nil, permissionsError(err)
	}

	err = h.engine.DeleteOAuthClient(ctx, request.ClientID)
	switch err {
	case nil, types.ErrOAuthClientNotFound:
	default:
		return nil, err
	}

	return DeleteOAuthClient200JSONResponse{Success: true}, nil
}
