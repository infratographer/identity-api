package httpsrv

import (
	"context"
	"net/http"

	"go.infratographer.com/identity-api/internal/crypto"
	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/x/gidx"
)

const (
	defaultTokenLength = 26

	actionOAuthClientCreate = "oauthclient_create"
	actionOAuthClientGet    = "oauthclient_get"
	actionOAuthClientDelete = "oauthclient_delete"
)

func (h *apiHandler) lookupOAuthClientWithResponse(ctx context.Context, id gidx.PrefixedID) (types.OAuthClient, error) {
	client, err := h.engine.LookupOAuthClientByID(ctx, id)

	switch err {
	case nil:
		return client, nil
	case types.ErrOAuthClientNotFound:
		return types.OAuthClient{}, errorWithStatus{
			status:  http.StatusNotFound,
			message: err.Error(),
		}
	default:
		return types.OAuthClient{}, err
	}
}

// Createoauthclient creates a client for a owner with a set name.
// This endpoint returns the OAuth client ID and secret that the client
// needs to provide to authenticate when requesting a token.
func (h *apiHandler) CreateOAuthClient(ctx context.Context, request CreateOAuthClientRequestObject) (CreateOAuthClientResponseObject, error) {
	if err := checkAccessWithResponse(ctx, request.OwnerID, actionOAuthClientCreate); err != nil {
		return nil, err
	}

	var newClient types.OAuthClient
	newClient.OwnerID = request.OwnerID
	newClient.Name = request.Body.Name

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
	client, err := h.lookupOAuthClientWithResponse(ctx, request.ClientID)
	if err != nil {
		return nil, err
	}

	if err := checkAccessWithResponse(ctx, client.OwnerID, actionOAuthClientGet); err != nil {
		return nil, err
	}

	apiType := client.ToV1OAuthClient()

	return GetOAuthClient200JSONResponse(apiType), nil
}

// DeleteOAuthClient removes the OAuth client.
func (h *apiHandler) DeleteOAuthClient(ctx context.Context, request DeleteOAuthClientRequestObject) (DeleteOAuthClientResponseObject, error) {
	client, err := h.lookupOAuthClientWithResponse(ctx, request.ClientID)
	if err != nil {
		return nil, err
	}

	if err := checkAccessWithResponse(ctx, client.OwnerID, actionOAuthClientDelete); err != nil {
		return nil, err
	}

	if err := h.engine.DeleteOAuthClient(ctx, request.ClientID); err != nil {
		return nil, err
	}

	return DeleteOAuthClient200JSONResponse{Success: true}, nil
}
