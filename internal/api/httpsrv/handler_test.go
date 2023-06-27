package httpsrv

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/gidx"
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

	ownerID := gidx.MustNewID("testten")
	issuerID := gidx.MustNewID("testiss")
	issuer := types.Issuer{
		OwnerID:       ownerID,
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
				OwnerID:       gidx.MustNewID("testten"),
				ID:            issuerID,
				Name:          "Example",
				URI:           "https://example.com/",
				JWKSURI:       "https://example.com/.well-known/jwks.json",
				ClaimMappings: mappingStrs,
			},
		},
	}

	store, err := storage.NewEngine(config, storage.WithMigrations(), storage.WithSeedData(seedData))
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
	}

	t.Run("CreateIssuer", func(t *testing.T) {
		t.Parallel()
		handler := apiHandler{
			engine: store,
		}

		createOp := &v1.CreateIssuer{
			ClaimMappings: &mappingStrs,
			JWKSURI:       "https://issuer.info/jwks.json",
			Name:          "Good issuer",
			URI:           "https://issuer.info/",
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := store.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := store.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[CreateIssuerRequestObject, CreateIssuerResponseObject]{
			{
				Name: "Success",
				Input: CreateIssuerRequestObject{
					OwnerID: ownerID,
					Body:    createOp,
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
					OwnerID: ownerID,
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
			engine: store,
		}

		testCases := []testingx.TestCase[GetIssuerByIDRequestObject, GetIssuerByIDResponseObject]{
			{
				Name: "Success",
				Input: GetIssuerByIDRequestObject{
					Id: issuerID,
				},
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[GetIssuerByIDResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuer := v1.Issuer{
						ID:            issuerID,
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
					Id: gidx.MustNewID("ntfound"),
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
			engine: store,
		}

		issuerID := gidx.MustNewID("testiss")

		issuer := types.Issuer{
			OwnerID:       ownerID,
			ID:            issuerID,
			Name:          "Example",
			URI:           "https://issuer.info/",
			JWKSURI:       "https://issuer.info/.well-known/jwks.json",
			ClaimMappings: mappings,
		}

		newName := "Better issuer"

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := store.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = store.CreateIssuer(ctx, issuer)
			if err != nil {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := store.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[UpdateIssuerRequestObject, UpdateIssuerResponseObject]{
			{
				Name: "Success",
				Input: UpdateIssuerRequestObject{
					Id: issuerID,
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
						ID:            issuerID,
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
					Id: gidx.MustNewID("ntfound"),
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
			engine: store,
		}

		issuerID := gidx.MustNewID("testiss")

		issuer := types.Issuer{
			OwnerID:       ownerID,
			ID:            issuerID,
			Name:          "Example",
			URI:           "https://issuer.info/",
			JWKSURI:       "https://issuer.info/.well-known/jwks.json",
			ClaimMappings: mappings,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := store.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = store.CreateIssuer(ctx, issuer)

			if !assert.NoError(t, err) {
				assert.FailNow(t, "error initializing issuer")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := store.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[DeleteIssuerRequestObject, DeleteIssuerResponseObject]{
			{
				Name: "Success",
				Input: DeleteIssuerRequestObject{
					Id: issuerID,
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

					_, err := store.GetIssuerByID(ctx, issuerID)
					assert.ErrorIs(t, err, types.ErrorIssuerNotFound)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name: "NotFound",
				Input: DeleteIssuerRequestObject{
					Id: gidx.MustNewID("ntfound"),
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

	t.Run("CreateOAuthClient", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := store.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := store.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		runFn := func(ctx context.Context, input CreateOAuthClientRequestObject) testingx.TestResult[CreateOAuthClientResponseObject] {
			resp, err := handler.CreateOAuthClient(ctx, input)

			result := testingx.TestResult[CreateOAuthClientResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testCases := []testingx.TestCase[CreateOAuthClientRequestObject, CreateOAuthClientResponseObject]{
			{
				Name: "Success",
				Input: CreateOAuthClientRequestObject{
					OwnerID: gidx.MustNewID("testten"),
					Body: &v1.CreateOAuthClientJSONRequestBody{
						Name:     "test-client",
						Audience: &[]string{"aud1", "aud2"},
					},
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[CreateOAuthClientResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, CreateOAuthClient200JSONResponse{}, res.Success)
					resp := v1.OAuthClient(res.Success.(CreateOAuthClient200JSONResponse))
					assert.NotEmpty(t, resp.ID)
					assert.NotEmpty(t, *resp.Secret)
					assert.Equal(t, []string{"aud1", "aud2"}, resp.Audience)
				},
				CleanupFn: cleanupFn,
			},
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetOAuthClient", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}
		client := types.OAuthClient{
			OwnerID:  ownerID,
			Name:     "Example",
			Secret:   "abc1234",
			Audience: []string{},
		}

		withStoredClients(t, store, &client)

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := store.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := store.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[GetOAuthClientRequestObject, GetOAuthClientResponseObject]{
			{
				Name: "NotFound",
				Input: GetOAuthClientRequestObject{
					ClientID: "",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[GetOAuthClientResponseObject]) {
					assert.IsType(t, errorWithStatus{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(errorWithStatus).status)
				},
			},
			{
				Name: "Success",
				Input: GetOAuthClientRequestObject{
					ClientID: client.ID,
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[GetOAuthClientResponseObject]) {
					assert.NoError(t, err)
					assert.IsType(t, GetOAuthClient200JSONResponse{}, res.Success)
					item := v1.OAuthClient(res.Success.(GetOAuthClient200JSONResponse))
					assert.Nil(t, item.Secret, "the secret field shouldn't be populated on a GET")
					assert.Equal(t, client.ID, item.ID)
					assert.Equal(t, client.Name, item.Name)
					assert.Equal(t, client.Audience, item.Audience)
				},
			},
		}

		runFn := func(ctx context.Context, input GetOAuthClientRequestObject) testingx.TestResult[GetOAuthClientResponseObject] {
			resp, err := handler.GetOAuthClient(ctx, input)

			result := testingx.TestResult[GetOAuthClientResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("DeleteOAuthClient", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}

		client := types.OAuthClient{
			OwnerID:  ownerID,
			Name:     "Example",
			Secret:   "abc1234",
			Audience: []string{},
		}

		withStoredClients(t, store, &client)

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := store.BeginContext(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := store.RollbackContext(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[DeleteOAuthClientRequestObject, DeleteOAuthClientResponseObject]{
			{
				Name: "Success",
				Input: DeleteOAuthClientRequestObject{
					ClientID: client.ID,
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[DeleteOAuthClientResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					resp, ok := result.Success.(DeleteOAuthClient200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for delete issuer response")
					}

					obsResp := v1.DeleteResponse(resp)

					expResp := v1.DeleteResponse{
						Success: true,
					}

					assert.Equal(t, expResp, obsResp)

					_, err := store.LookupOAuthClientByID(ctx, client.ID)
					assert.ErrorIs(t, types.ErrOAuthClientNotFound, err)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name: "NotFound",
				Input: DeleteOAuthClientRequestObject{
					ClientID: gidx.MustNewID("ntfound"),
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, result testingx.TestResult[DeleteOAuthClientResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					resp, ok := result.Success.(DeleteOAuthClient200JSONResponse)
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

		runFn := func(ctx context.Context, input DeleteOAuthClientRequestObject) testingx.TestResult[DeleteOAuthClientResponseObject] {
			resp, err := handler.DeleteOAuthClient(ctx, input)

			result := testingx.TestResult[DeleteOAuthClientResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})
}

func withStoredClients(t *testing.T, s storage.Engine, clients ...*types.OAuthClient) {
	seedCtx, err := s.BeginContext(context.Background())
	if err != nil {
		assert.FailNow(t, "failed to begin context")
	}

	for _, c := range clients {
		client := c
		*client, err = s.CreateOAuthClient(seedCtx, *client)

		if !assert.NoError(t, err) {
			assert.FailNow(t, "error initializing oauth client")
		}
	}

	if err := s.CommitContext(seedCtx); err != nil {
		assert.FailNow(t, "error committing seed clients")
	}
}
