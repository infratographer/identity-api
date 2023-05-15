package httpsrv

import (
	"context"
	"net/http"

	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/gidx"
)

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

	id, err := gidx.NewID(types.IdentityIssuerIDPrefix)
	if err != nil {
		err = errorWithStatus{
			status:  http.StatusInternalServerError,
			message: "failed to generate new id",
		}

		return nil, err
	}

	issuerToCreate := types.Issuer{
		TenantID:      tenantID,
		ID:            id,
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
	iss, err := h.engine.GetIssuerByID(ctx, req.Id)
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

	issuer, err := h.engine.UpdateIssuer(ctx, req.Id, update)
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
	err := h.engine.DeleteIssuer(ctx, req.Id)
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
