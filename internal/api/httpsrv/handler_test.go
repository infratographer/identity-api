package httpsrv

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-manager-sts/internal/storage"
	"go.infratographer.com/identity-manager-sts/internal/testingx"
	"go.infratographer.com/identity-manager-sts/internal/types"
	v1 "go.infratographer.com/identity-manager-sts/pkg/api/v1"
)

func TestAPIHandler(t *testing.T) {
	mappingStrs := map[string]string{
		"foo": "123",
	}

	mappings, err := types.NewClaimsMapping(mappingStrs)
	if err != nil {
		panic(err)
	}

	tenantID := "56a95c1b-33f8-4def-8b6d-ca9fe6976170"
	tenantUUID := uuid.MustParse(tenantID)
	issuerID := "e495a393-ae79-4a02-a78d-9798c7d9d252"
	issuerUUID := uuid.MustParse(issuerID)
	issuer := types.Issuer{
		TenantID:      tenantID,
		ID:            issuerID,
		Name:          "Example",
		URI:           "https://example.com/",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: mappings,
	}

	config := storage.Config{
		Type: storage.EngineTypeMemory,
		SeedData: storage.SeedData{
			Issuers: []storage.SeedIssuer{
				{
					TenantID:      tenantID,
					ID:            issuer.ID,
					Name:          issuer.Name,
					URI:           issuer.URI,
					JWKSURI:       issuer.JWKSURI,
					ClaimMappings: mappingStrs,
				},
			},
		},
	}

	issSvc, err := storage.NewEngine(config)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	t.Run("CreateIssuer", func(t *testing.T) {
		t.Parallel()
		handler := apiHandler{
			engine: issSvc,
		}

		createOp := &v1.CreateIssuer{
			ClaimMappings: &mappingStrs,
			JWKSURI:       "https://example.com/jwks.json",
			Name:          "Good issuer",
			URI:           "https://example.com/",
		}

		testCases := []testingx.TestCase[CreateIssuerRequestObject, CreateIssuerResponseObject]{
			{
				Name: "Success",
				Input: CreateIssuerRequestObject{
					TenantID: tenantUUID,
					Body:     createOp,
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[CreateIssuerResponseObject]) {
					// Just stop if we failed
					if !assert.NoError(t, result.Err) {
						return
					}

					resp, ok := result.Success.(CreateIssuer200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for create issuer response")
					}

					obsIssuer := v1.Issuer(resp)

					expIssuer := v1.Issuer{
						ID:            obsIssuer.ID,
						ClaimMappings: *createOp.ClaimMappings,
						JWKSURI:       createOp.JWKSURI,
						Name:          createOp.Name,
						URI:           createOp.URI,
					}

					assert.Equal(t, expIssuer, obsIssuer)
				},
			},
			{
				Name: "CELError",
				Input: CreateIssuerRequestObject{
					TenantID: tenantUUID,
					Body: &v1.CreateIssuer{
						ClaimMappings: &map[string]string{
							"bad": "'123",
						},
						JWKSURI: "https://bad.info/jwks.json",
						Name:    "Bad issuer",
						URI:     "https://bad.info/",
					},
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[CreateIssuerResponseObject]) {
					// We expect a 400 here, not a 500
					if !assert.NoError(t, result.Err) {
						return
					}

					obsResp, ok := result.Success.(CreateIssuer400JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for create issuer response")
					}

					expResp := CreateIssuer400JSONResponse{
						Errors: []string{
							"error parsing CEL expression",
						},
					}

					assert.Equal(t, expResp, obsResp)
				},
			},
		}

		runFn := func(ctx context.Context, input CreateIssuerRequestObject) testingx.TestResult[CreateIssuerResponseObject] {
			resp, err := handler.CreateIssuer(ctx, input)

			result := testingx.TestResult[CreateIssuerResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetIssuer", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: issSvc,
		}

		testCases := []testingx.TestCase[GetIssuerByIDRequestObject, GetIssuerByIDResponseObject]{
			{
				Name: "Success",
				Input: GetIssuerByIDRequestObject{
					Id: issuerUUID,
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[GetIssuerByIDResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuer := v1.Issuer{
						ID:            issuerUUID,
						ClaimMappings: mappingStrs,
						JWKSURI:       issuer.JWKSURI,
						Name:          issuer.Name,
						URI:           issuer.URI,
					}

					resp, ok := result.Success.(GetIssuerByID200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get issuer response")
					}

					obsIssuer := v1.Issuer(resp)

					assert.Equal(t, expIssuer, obsIssuer)
				},
			},
			{
				Name: "NotFound",
				Input: GetIssuerByIDRequestObject{
					Id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[GetIssuerByIDResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					obsResp, ok := result.Success.(GetIssuerByID404JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get issuer response")
					}

					expResp := GetIssuerByID404JSONResponse{
						Errors: []string{
							"not found",
						},
					}

					assert.Equal(t, expResp, obsResp)
				},
			},
		}

		runFn := func(ctx context.Context, input GetIssuerByIDRequestObject) testingx.TestResult[GetIssuerByIDResponseObject] {
			resp, err := handler.GetIssuerByID(ctx, input)

			result := testingx.TestResult[GetIssuerByIDResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("UpdateIssuer", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: issSvc,
		}

		issuerID := "53dcdc2a-94e7-44d1-97f5-ce7ad136b698"
		issuerUUID := uuid.MustParse(issuerID)

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            issuerID,
			Name:          "Example",
			URI:           "https://example.com/",
			JWKSURI:       "https://example.com/.well-known/jwks.json",
			ClaimMappings: mappings,
		}

		if _, err := issSvc.CreateIssuer(context.Background(), issuer); err != nil {
			assert.FailNow(t, "error initializing issuer")
		}

		newName := "Better issuer"

		testCases := []testingx.TestCase[UpdateIssuerRequestObject, UpdateIssuerResponseObject]{
			{
				Name: "Success",
				Input: UpdateIssuerRequestObject{
					Id: issuerUUID,
					Body: &v1.IssuerUpdate{
						Name: &newName,
					},
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[UpdateIssuerResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuer := v1.Issuer{
						ID:            issuerUUID,
						ClaimMappings: mappingStrs,
						JWKSURI:       issuer.JWKSURI,
						Name:          newName,
						URI:           issuer.URI,
					}

					resp, ok := result.Success.(UpdateIssuer200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for update issuer response")
					}

					obsIssuer := v1.Issuer(resp)

					assert.Equal(t, expIssuer, obsIssuer)
				},
			},
			{
				Name: "NotFound",
				Input: UpdateIssuerRequestObject{
					Id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Body: &v1.IssuerUpdate{
						Name: &newName,
					},
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[UpdateIssuerResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					obsResp, ok := result.Success.(UpdateIssuer404JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for update issuer response")
					}

					expResp := UpdateIssuer404JSONResponse{
						Errors: []string{
							"not found",
						},
					}

					assert.Equal(t, expResp, obsResp)
				},
			},
		}

		runFn := func(ctx context.Context, input UpdateIssuerRequestObject) testingx.TestResult[UpdateIssuerResponseObject] {
			resp, err := handler.UpdateIssuer(ctx, input)

			result := testingx.TestResult[UpdateIssuerResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("DeleteIssuer", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: issSvc,
		}

		issuerID := "c8d6458e-2524-4cf9-9e8d-a3f94b114b74"
		issuerUUID := uuid.MustParse(issuerID)

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            issuerID,
			Name:          "Example",
			URI:           "https://example.com/",
			JWKSURI:       "https://example.com/.well-known/jwks.json",
			ClaimMappings: mappings,
		}

		if _, err := issSvc.CreateIssuer(context.Background(), issuer); err != nil {
			assert.FailNow(t, "error initializing issuer")
		}

		testCases := []testingx.TestCase[DeleteIssuerRequestObject, DeleteIssuerResponseObject]{
			{
				Name: "Success",
				Input: DeleteIssuerRequestObject{
					Id: issuerUUID,
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[DeleteIssuerResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					resp, ok := result.Success.(DeleteIssuer200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for delete issuer response")
					}

					obsResp := v1.DeleteResponse(resp)

					expResp := v1.DeleteResponse{
						Success: true,
					}

					assert.Equal(t, expResp, obsResp)
				},
			},
			{
				Name: "NotFound",
				Input: DeleteIssuerRequestObject{
					Id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[DeleteIssuerResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					obsResp, ok := result.Success.(DeleteIssuer404JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for delete issuer response")
					}

					expResp := DeleteIssuer404JSONResponse{
						Errors: []string{
							"not found",
						},
					}

					assert.Equal(t, expResp, obsResp)
				},
			},
		}

		runFn := func(ctx context.Context, input DeleteIssuerRequestObject) testingx.TestResult[DeleteIssuerResponseObject] {
			resp, err := handler.DeleteIssuer(ctx, input)

			result := testingx.TestResult[DeleteIssuerResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})
}
