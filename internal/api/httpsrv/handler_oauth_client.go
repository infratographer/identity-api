package httpsrv

import (
	"context"
	"net/http"

	"go.infratographer.com/identity-api/internal/crypto"
	"go.infratographer.com/identity-api/internal/types"
)

const defaultTokenLength = 26

// CreateOAuthClient creates a client for a tenant with a set name.
// This endpoint returns the OAuth client ID and secret that the client
// needs to provide to authenticate when requesting a token.
func (h *apiHandler) CreateOAuthClient(ctx context.Context, request CreateOAuthClientRequestObject) (CreateOAuthClientResponseObject, error) {
	var newClient types.OAuthClient
	newClient.TenantID = request.TenantID
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

	apiType := client.ToV1OAuthClient()

	return GetOAuthClient200JSONResponse(apiType), nil
}

// DeleteOAuthClient removes the OAuth client.
func (h *apiHandler) DeleteOAuthClient(ctx context.Context, request DeleteOAuthClientRequestObject) (DeleteOAuthClientResponseObject, error) {
	err := h.engine.DeleteOAuthClient(ctx, request.ClientID)
	switch err {
	case nil, types.ErrOAuthClientNotFound:
	default:
		return nil, err
	}

	return DeleteOAuthClient200JSONResponse{Success: true}, nil
}
