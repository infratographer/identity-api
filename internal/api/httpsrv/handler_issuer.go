package httpsrv

import (
	"context"
	"net/http"

	"go.infratographer.com/permissions-api/pkg/permissions"
	"go.infratographer.com/x/gidx"

	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

const (
	actionIssuerCreate = "iam_issuer_create"
	actionIssuerUpdate = "iam_issuer_update"
	actionIssuerDelete = "iam_issuer_delete"
	actionIssuerGet    = "iam_issuer_get"
	actionIssuerList   = "iam_issuer_list"
)

func (h *apiHandler) CreateIssuer(ctx context.Context, req CreateIssuerRequestObject) (CreateIssuerResponseObject, error) {
	ownerID := req.OwnerID
	createOp := req.Body

	if err := permissions.CheckAccess(ctx, ownerID, actionIssuerCreate); err != nil {
		return nil, permissionsError(err)
	}

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
		OwnerID:       ownerID,
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

	if err := permissions.CheckAccess(ctx, iss.OwnerID, actionIssuerGet); err != nil {
		return nil, permissionsError(err)
	}

	out, err := iss.ToV1Issuer()
	if err != nil {
		return nil, err
	}

	return GetIssuerByID200JSONResponse(out), nil
}

func (h *apiHandler) ListOwnerIssuers(ctx context.Context, req ListOwnerIssuersRequestObject) (ListOwnerIssuersResponseObject, error) {
	if err := permissions.CheckAccess(ctx, req.OwnerID, actionIssuerList); err != nil {
		return nil, permissionsError(err)
	}

	iss, err := h.engine.GetOwnerIssuers(ctx, req.OwnerID, req.Params)
	if err != nil {
		return nil, err
	}

	issuers, err := iss.ToV1Issuers()
	if err != nil {
		return nil, err
	}

	collection := v1.IssuerCollection{
		Issuers:    issuers,
		Pagination: v1.Pagination{},
	}

	if err := req.Params.SetPagination(&collection); err != nil {
		return nil, err
	}

	out := IssuerCollectionJSONResponse(collection)

	return ListOwnerIssuers200JSONResponse{out}, nil
}

func (h *apiHandler) UpdateIssuer(ctx context.Context, req UpdateIssuerRequestObject) (UpdateIssuerResponseObject, error) {
	// We must fetch the issuer to retrieve the owner so we may check for permission to update.
	iss, err := h.engine.GetIssuerByID(ctx, req.Id)
	switch err {
	case nil:
	case types.ErrorIssuerNotFound:
		return nil, errorNotFound
	default:
		return nil, err
	}

	if err := permissions.CheckAccess(ctx, iss.OwnerID, actionIssuerUpdate); err != nil {
		return nil, permissionsError(err)
	}

	updateOp := req.Body

	var claimsMapping types.ClaimsMapping

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
	// We must fetch the issuer to retrieve the owner so we may check for permission to delete.
	iss, err := h.engine.GetIssuerByID(ctx, req.Id)
	switch err {
	case nil:
	case types.ErrorIssuerNotFound:
		return nil, errorNotFound
	default:
		return nil, err
	}

	if err := permissions.CheckAccess(ctx, iss.OwnerID, actionIssuerDelete); err != nil {
		return nil, permissionsError(err)
	}

	err = h.engine.DeleteIssuer(ctx, req.Id)
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
