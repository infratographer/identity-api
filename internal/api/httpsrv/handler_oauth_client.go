package httpsrv

import (
	"context"

	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

func (h *apiHandler) DeleteOAuthClient(ctx context.Context, request DeleteOAuthClientRequestObject) (DeleteOAuthClientResponseObject, error) {
	return DeleteOAuthClient200JSONResponse{Success: true}, nil
}

func (h *apiHandler) GetOAuthClient(ctx context.Context, request GetOAuthClientRequestObject) (GetOAuthClientResponseObject, error) {
	var out v1.OAuthClient

	return GetOAuthClient200JSONResponse(out), nil
}

func (h *apiHandler) CreateOAuthClient(ctx context.Context, reqeust CreateOAuthClientRequestObject) (CreateOAuthClientResponseObject, error) {
	var out v1.OAuthClient

	return CreateOAuthClient200JSONResponse(out), nil
}
