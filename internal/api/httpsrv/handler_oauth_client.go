package httpsrv

import (
	"context"

	"go.infratographer.com/identity-api/internal/crypto"
	"go.infratographer.com/identity-api/internal/types"
)

// CreateOAuthClient creates a client for a tenant with a set name.
// This endpoint returns the OAuth client ID and secret that the client
// needs to provide to authenticate when requesting a token.
func (h *apiHandler) CreateOAuthClient(ctx context.Context, request CreateOAuthClientRequestObject) (CreateOAuthClientResponseObject, error) {
	var newClient types.OAuthClient
	newClient.TenantID = request.TenantID.String()

	if request.Body.Audience != nil {
		newClient.Audience = *request.Body.Audience
	}

	secret, err := crypto.GenerateSecureToken()
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
	client, err := h.engine.LookupOAuthClientByID(ctx, request.ClientID.String())
	switch err {
	case nil:
	case types.ErrOAuthClientNotFound:
		return GetOAuthClient404JSONResponse{}, err
	default:
		return nil, err
	}

	apiType := client.ToV1OAuthClient()

	// Don't return the secret hash
	var emptySecret string
	apiType.Secret = &emptySecret

	return GetOAuthClient200JSONResponse(apiType), nil
}

// DeleteOAuthClient removes the OAuth client.
func (h *apiHandler) DeleteOAuthClient(ctx context.Context, request DeleteOAuthClientRequestObject) (DeleteOAuthClientResponseObject, error) {
	err := h.engine.DeleteOAuthClient(ctx, request.ClientID.String())
	switch err {
	case nil, types.ErrOAuthClientNotFound:
	default:
		return nil, err
	}

	return DeleteOAuthClient200JSONResponse{Success: true}, nil
}
