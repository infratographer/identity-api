package httpsrv

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/crdbx"
)

func TestAPIHandler(t *testing.T) {
	t.Parallel()

	testServer, err := storage.InMemoryCRDB()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	err = testServer.Start()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	t.Cleanup(func() {
		testServer.Stop()
	})

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

	config := crdbx.Config{
		URI: testServer.PGURL().String(),
	}

	seedData := storage.SeedData{
		Issuers: []storage.SeedIssuer{
			{
				TenantID:      "b8bfd705-b768-47a4-85a0-fe006f5bcfca",
				ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
				Name:          "Example",
				URI:           "https://example.com/",
				JWKSURI:       "https://example.com/.well-known/jwks.json",
				ClaimMappings: mappingStrs,
			},
		},
	}

	issSvc, err := storage.NewEngine(config, storage.WithMigrations(), storage.WithSeedData(seedData))
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
			JWKSURI:       "https://issuer.info/jwks.json",
			Name:          "Good issuer",
			URI:           "https://issuer.info/",
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := issSvc.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := issSvc.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[CreateIssuerRequestObject, CreateIssuerResponseObject]{
			{
				Name: "Success",
				Input: CreateIssuerRequestObject{
					TenantID: tenantUUID,
					Body:     createOp,
				},
				SetupFn: setupFn,
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
				CleanupFn: cleanupFn,
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
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[CreateIssuerResponseObject]) {
					expErr := errorWithStatus{
						status:  http.StatusBadRequest,
						message: "error parsing CEL expression",
					}

					assert.ErrorIs(t, expErr, result.Err)
				},
				CleanupFn: cleanupFn,
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
					assert.ErrorIs(t, errorNotFound, result.Err)
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
			URI:           "https://issuer.info/",
			JWKSURI:       "https://issuer.info/.well-known/jwks.json",
			ClaimMappings: mappings,
		}

		newName := "Better issuer"

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := issSvc.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = issSvc.CreateIssuer(ctx, issuer)
			if err != nil {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := issSvc.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[UpdateIssuerRequestObject, UpdateIssuerResponseObject]{
			{
				Name: "Success",
				Input: UpdateIssuerRequestObject{
					Id: issuerUUID,
					Body: &v1.IssuerUpdate{
						Name: &newName,
					},
				},
				SetupFn: setupFn,
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
				CleanupFn: cleanupFn,
			},
			{
				Name: "NotFound",
				Input: UpdateIssuerRequestObject{
					Id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Body: &v1.IssuerUpdate{
						Name: &newName,
					},
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[UpdateIssuerResponseObject]) {
					assert.ErrorIs(t, errorNotFound, result.Err)
				},
				CleanupFn: cleanupFn,
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
			URI:           "https://issuer.info/",
			JWKSURI:       "https://issuer.info/.well-known/jwks.json",
			ClaimMappings: mappings,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := issSvc.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = issSvc.CreateIssuer(ctx, issuer)

			if !assert.NoError(t, err) {
				assert.FailNow(t, "error initializing issuer")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := issSvc.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[DeleteIssuerRequestObject, DeleteIssuerResponseObject]{
			{
				Name: "Success",
				Input: DeleteIssuerRequestObject{
					Id: issuerUUID,
				},
				SetupFn: setupFn,
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

					_, err := issSvc.GetIssuerByID(ctx, issuerID)
					assert.ErrorIs(t, err, types.ErrorIssuerNotFound)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name: "NotFound",
				Input: DeleteIssuerRequestObject{
					Id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				SetupFn: setupFn,
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
				CleanupFn: cleanupFn,
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
