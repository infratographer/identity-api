package httpsrv

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.infratographer.com/permissions-api/pkg/permissions"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/gidx"

	pagination "go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

func ctxPermsAllow(ctx context.Context) context.Context {
	return context.WithValue(ctx, permissions.CheckerCtxKey, permissions.DefaultAllowChecker)
}

func ptr[T any](v T) *T {
	return &v
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

//nolint:gocyclo
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[CreateIssuerResponseObject]) {
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[CreateIssuerResponseObject]) {
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetIssuerByIDResponseObject]) {
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetIssuerByIDResponseObject]) {
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
	})

	t.Run("ListIssuers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}

		var (
			issOwnerID = gidx.PrefixedID("testten-" + t.Name())
			iss1       = types.Issuer{
				OwnerID: issOwnerID,
				ID:      gidx.PrefixedID("testiss-" + t.Name() + "-1"),
				Name:    t.Name() + "-1",
				URI:     "https://" + t.Name() + "-1.example.com/",
				JWKSURI: "https://" + t.Name() + "-1.example.com/.well-known/jwks.json",
			}
			v1iss1 = must(iss1.ToV1Issuer())

			iss2 = types.Issuer{
				OwnerID: issOwnerID,
				ID:      gidx.PrefixedID("testiss-" + t.Name() + "-2"),
				Name:    t.Name() + "-2",
				URI:     "https://" + t.Name() + "-2.example.com/",
				JWKSURI: "https://" + t.Name() + "-2.example.com/.well-known/jwks.json",
			}
			v1iss2 = must(iss2.ToV1Issuer())

			iss3 = types.Issuer{
				OwnerID: gidx.MustNewID("testten"),
				ID:      gidx.PrefixedID("testiss-" + t.Name() + "-3"),
				Name:    t.Name() + "-3",
				URI:     "https://" + t.Name() + "-3.example.com/",
				JWKSURI: "https://" + t.Name() + "-3.example.com/.well-known/jwks.json",
			}
		)

		withStoredIssuers(t, store, &iss1, &iss2, &iss3)

		testCases := []testingx.TestCase[ListOwnerIssuersRequestObject, ListOwnerIssuersResponseObject]{
			{
				Name: "Success (Default Pagination)",
				Input: ListOwnerIssuersRequestObject{
					OwnerID: issOwnerID,
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[ListOwnerIssuersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuers := []v1.Issuer{v1iss1, v1iss2}

					expPagination := v1.Pagination{
						Limit: 10,
					}

					resp, ok := result.Success.(ListOwnerIssuers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get issuer response")
					}

					assert.Equal(t, expIssuers, resp.Issuers, "unexpected issuers returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with limit",
				Input: ListOwnerIssuersRequestObject{
					OwnerID: issOwnerID,
					Params: v1.ListOwnerIssuersParams{
						Limit: ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[ListOwnerIssuersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuers := []v1.Issuer{v1iss1}

					expPagination := v1.Pagination{
						Limit: 1,
						Next:  pagination.MustNewCursor("id", iss1.ID.String()),
					}

					resp, ok := result.Success.(ListOwnerIssuers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get issuer response")
					}

					assert.Equal(t, expIssuers, resp.Issuers, "unexpected issuers returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with cursor",
				Input: ListOwnerIssuersRequestObject{
					OwnerID: issOwnerID,
					Params: v1.ListOwnerIssuersParams{
						Cursor: pagination.MustNewCursor("id", iss1.ID.String()),
						Limit:  ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[ListOwnerIssuersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuers := []v1.Issuer{v1iss2}

					expPagination := v1.Pagination{
						Limit: 1,
						Next:  pagination.MustNewCursor("id", iss2.ID.String()),
					}

					resp, ok := result.Success.(ListOwnerIssuers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get issuer response")
					}

					assert.Equal(t, expIssuers, resp.Issuers, "unexpected issuers returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with cursor end of results",
				Input: ListOwnerIssuersRequestObject{
					OwnerID: issOwnerID,
					Params: v1.ListOwnerIssuersParams{
						Cursor: pagination.MustNewCursor("id", v1iss2.ID.String()),
						Limit:  ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[ListOwnerIssuersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expIssuers := []v1.Issuer{}

					expPagination := v1.Pagination{
						Limit: 1,
					}

					resp, ok := result.Success.(ListOwnerIssuers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get issuer response")
					}

					assert.Equal(t, expIssuers, resp.Issuers, "unexpected issuers returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
		}

		runFn := func(ctx context.Context, input ListOwnerIssuersRequestObject) testingx.TestResult[ListOwnerIssuersResponseObject] {
			ctx = pagination.AsOfSystemTime(ctx, "")

			resp, err := handler.ListOwnerIssuers(ctx, input)

			result := testingx.TestResult[ListOwnerIssuersResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[UpdateIssuerResponseObject]) {
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[UpdateIssuerResponseObject]) {
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[DeleteIssuerResponseObject]) {
					if !assert.Error(t, result.Err) {
						return
					}

					assert.ErrorIs(t, result.Err, errorNotFound)
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
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
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[CreateOAuthClientResponseObject]) {
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
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
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetOAuthClientResponseObject]) {
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
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetOAuthClientResponseObject]) {
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
	})

	t.Run("ListOAuthClients", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}

		var (
			cliOwnerID = gidx.PrefixedID("testten-" + t.Name())
			cli1       = types.OAuthClient{
				OwnerID: cliOwnerID,
				Name:    t.Name() + "-1",
				Secret:  "abc1234",
			}

			cli2 = types.OAuthClient{
				OwnerID: cliOwnerID,
				Name:    t.Name() + "-2",
				Secret:  "def4567",
			}

			cli3 = types.OAuthClient{
				OwnerID: gidx.MustNewID("testten"),
				Name:    t.Name() + "-3",
				Secret:  "ghi7890",
			}
		)

		withStoredClients(t, store, &cli1, &cli2, &cli3)

		// fetch the stored clients so we know the order to be able to assert expected results
		clients, err := store.GetOwnerOAuthClients(pagination.AsOfSystemTime(context.Background(), ""), cliOwnerID, pagination.Pagination{})
		require.NoError(t, err, "unexpected error fetching clients")

		require.Len(t, clients, 2, "expected two clients to exist")

		cli1, cli2 = clients[0], clients[1]
		v1cli1, v1cli2 := cli1.ToV1OAuthClient(), cli2.ToV1OAuthClient()

		testCases := []testingx.TestCase[GetOwnerOAuthClientsRequestObject, GetOwnerOAuthClientsResponseObject]{
			{
				Name: "Success (Default Pagination)",
				Input: GetOwnerOAuthClientsRequestObject{
					OwnerID: cliOwnerID,
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetOwnerOAuthClientsResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expClients := []v1.OAuthClient{v1cli1, v1cli2}

					expPagination := v1.Pagination{
						Limit: 10,
					}

					resp, ok := result.Success.(GetOwnerOAuthClients200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get OAuth client response")
					}

					assert.Equal(t, expClients, resp.Clients, "unexpected OAuth clients returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with limit",
				Input: GetOwnerOAuthClientsRequestObject{
					OwnerID: cliOwnerID,
					Params: v1.GetOwnerOAuthClientsParams{
						Limit: ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetOwnerOAuthClientsResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expClients := []v1.OAuthClient{v1cli1}

					expPagination := v1.Pagination{
						Limit: 1,
						Next:  pagination.MustNewCursor("id", cli1.ID.String()),
					}

					resp, ok := result.Success.(GetOwnerOAuthClients200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get OAuth client response")
					}

					assert.Equal(t, expClients, resp.Clients, "unexpected OAuth clients returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with cursor",
				Input: GetOwnerOAuthClientsRequestObject{
					OwnerID: cliOwnerID,
					Params: v1.GetOwnerOAuthClientsParams{
						Cursor: pagination.MustNewCursor("id", cli1.ID.String()),
						Limit:  ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetOwnerOAuthClientsResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expClients := []v1.OAuthClient{v1cli2}

					expPagination := v1.Pagination{
						Limit: 1,
						Next:  pagination.MustNewCursor("id", cli2.ID.String()),
					}

					resp, ok := result.Success.(GetOwnerOAuthClients200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get OAuth client response")
					}

					assert.Equal(t, expClients, resp.Clients, "unexpected OAuth clients returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with cursor end of results",
				Input: GetOwnerOAuthClientsRequestObject{
					OwnerID: cliOwnerID,
					Params: v1.GetOwnerOAuthClientsParams{
						Cursor: pagination.MustNewCursor("id", v1cli2.ID.String()),
						Limit:  ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetOwnerOAuthClientsResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expClients := []v1.OAuthClient{}

					expPagination := v1.Pagination{
						Limit: 1,
					}

					resp, ok := result.Success.(GetOwnerOAuthClients200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get OAuth client response")
					}

					assert.Equal(t, expClients, resp.Clients, "unexpected OAuth clients returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
		}

		runFn := func(ctx context.Context, input GetOwnerOAuthClientsRequestObject) testingx.TestResult[GetOwnerOAuthClientsResponseObject] {
			ctx = pagination.AsOfSystemTime(ctx, "")

			resp, err := handler.GetOwnerOAuthClients(ctx, input)

			result := testingx.TestResult[GetOwnerOAuthClientsResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
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
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[DeleteOAuthClientResponseObject]) {
					if !assert.Error(t, result.Err) {
						return
					}

					err, ok := result.Err.(errorWithStatus)
					if !ok {
						assert.FailNow(t, "unexpected error type returned", result.Err)
					}

					assert.Equal(t, types.ErrOAuthClientNotFound.Error(), err.Error())
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

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
	})

	t.Run("GetUser", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}
		issuer := types.Issuer{
			OwnerID: ownerID,
			Name:    "Example",
			URI:     "https://example2.com/",
			JWKSURI: "https://example2.com/.well-known/jwks.json",
		}

		withStoredIssuers(t, store, &issuer)

		userInfo := types.UserInfo{
			Name:    t.Name(),
			Email:   t.Name() + "@example.com",
			Issuer:  issuer.URI,
			Subject: t.Name() + "Test",
		}

		withStoredUsers(t, store, &userInfo)

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

		testCases := []testingx.TestCase[GetUserByIDRequestObject, GetUserByIDResponseObject]{
			{
				Name: "NotFound",
				Input: GetUserByIDRequestObject{
					UserID: "",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetUserByIDResponseObject]) {
					assert.IsType(t, errorWithStatus{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(errorWithStatus).status)
				},
			},
			{
				Name: "Success",
				Input: GetUserByIDRequestObject{
					UserID: userInfo.ID,
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetUserByIDResponseObject]) {
					assert.NoError(t, err)
					assert.IsType(t, GetUserByID200JSONResponse{}, res.Success)
					item := v1.User(res.Success.(GetUserByID200JSONResponse))
					assert.Equal(t, userInfo.ID, item.ID)
					assert.Equal(t, userInfo.Name, *item.Name)
					assert.Equal(t, userInfo.Email, *item.Email)
					assert.Equal(t, userInfo.Issuer, item.Issuer)
					assert.Equal(t, userInfo.Subject, item.Subject)
				},
			},
		}

		runFn := func(ctx context.Context, input GetUserByIDRequestObject) testingx.TestResult[GetUserByIDResponseObject] {
			resp, err := handler.GetUserByID(ctx, input)

			result := testingx.TestResult[GetUserByIDResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
	})

	t.Run("ListIssuerUsers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine: store,
		}

		var (
			issOwnerID = gidx.PrefixedID("testten-" + t.Name())

			iss1dom = t.Name() + "-1.example.com"
			iss1    = types.Issuer{
				OwnerID: issOwnerID,
				ID:      gidx.MustNewID("testiss"),
				Name:    t.Name() + "-1",
				URI:     "https://" + iss1dom + "/",
				JWKSURI: "https://" + iss1dom + "/.well-known/jwks.json",
			}

			iss2dom = t.Name() + "-2.example.com"
			iss2    = types.Issuer{
				OwnerID: issOwnerID,
				ID:      gidx.MustNewID("testiss"),
				Name:    t.Name() + "-2",
				URI:     "https://" + iss2dom + "/",
				JWKSURI: "https://" + iss2dom + "/.well-known/jwks.json",
			}
		)

		withStoredIssuers(t, store, &iss1, &iss2)

		var (
			usr1 = types.UserInfo{
				Name:    t.Name() + "-1.1",
				Email:   t.Name() + "-1.1@" + iss1dom,
				Issuer:  iss1.URI,
				Subject: t.Name() + "-1.1 Test",
			}
			usr2 = types.UserInfo{
				Name:    t.Name() + "-1.2",
				Email:   t.Name() + "-1.2@" + iss1dom,
				Issuer:  iss1.URI,
				Subject: t.Name() + "-1.2 Test",
			}
			usr3 = types.UserInfo{
				Name:    t.Name() + "-2.1",
				Email:   t.Name() + "-2.1@" + iss2dom,
				Issuer:  iss2.URI,
				Subject: t.Name() + "-2.1 Test",
			}
		)

		withStoredUsers(t, store, &usr1, &usr2, &usr3)

		// fetch the stored users so we know the order to be able to assert expected results
		users, err := store.LookupUserInfosByIssuerID(pagination.AsOfSystemTime(context.Background(), ""), iss1.ID, pagination.Pagination{})
		require.NoError(t, err, "unexpected error fetching users")

		require.Len(t, users, 2, "expected two users to exist")

		usr1, usr2 = users[0], users[1]
		v1usr1, v1usr2 := must(usr1.ToV1User()), must(usr2.ToV1User())

		testCases := []testingx.TestCase[GetIssuerUsersRequestObject, GetIssuerUsersResponseObject]{
			{
				Name: "Success (Default Pagination)",
				Input: GetIssuerUsersRequestObject{
					IssuerID: iss1.ID,
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetIssuerUsersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expUsers := []v1.User{v1usr1, v1usr2}

					expPagination := v1.Pagination{
						Limit: 10,
					}

					resp, ok := result.Success.(GetIssuerUsers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get users response")
					}

					assert.Equal(t, expUsers, resp.Users, "unexpected users returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with limit",
				Input: GetIssuerUsersRequestObject{
					IssuerID: iss1.ID,
					Params: v1.GetIssuerUsersParams{
						Limit: ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetIssuerUsersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expUsers := []v1.User{v1usr1}

					expPagination := v1.Pagination{
						Limit: 1,
						Next:  pagination.MustNewCursor("id", usr1.ID.String()),
					}

					resp, ok := result.Success.(GetIssuerUsers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get users response")
					}

					assert.Equal(t, expUsers, resp.Users, "unexpected users returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with cursor",
				Input: GetIssuerUsersRequestObject{
					IssuerID: iss1.ID,
					Params: v1.GetIssuerUsersParams{
						Cursor: pagination.MustNewCursor("id", usr1.ID.String()),
						Limit:  ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetIssuerUsersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expUsers := []v1.User{v1usr2}

					expPagination := v1.Pagination{
						Limit: 1,
						Next:  pagination.MustNewCursor("id", usr2.ID.String()),
					}

					resp, ok := result.Success.(GetIssuerUsers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get users response")
					}

					assert.Equal(t, expUsers, resp.Users, "unexpected users returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
			{
				Name: "Success with cursor end of results",
				Input: GetIssuerUsersRequestObject{
					IssuerID: iss1.ID,
					Params: v1.GetIssuerUsersParams{
						Cursor: pagination.MustNewCursor("id", v1usr2.ID.String()),
						Limit:  ptr(1),
					},
				},
				CheckFn: func(_ context.Context, t *testing.T, result testingx.TestResult[GetIssuerUsersResponseObject]) {
					if !assert.NoError(t, result.Err) {
						return
					}

					expUsers := []v1.User{}

					expPagination := v1.Pagination{
						Limit: 1,
					}

					resp, ok := result.Success.(GetIssuerUsers200JSONResponse)
					if !ok {
						assert.FailNow(t, "unexpected result type for get users response")
					}

					assert.Equal(t, expUsers, resp.Users, "unexpected users returned")
					assert.Equal(t, expPagination, resp.Pagination, "unexpected pagination returned")
				},
			},
		}

		runFn := func(ctx context.Context, input GetIssuerUsersRequestObject) testingx.TestResult[GetIssuerUsersResponseObject] {
			ctx = pagination.AsOfSystemTime(ctx, "")

			resp, err := handler.GetIssuerUsers(ctx, input)

			result := testingx.TestResult[GetIssuerUsersResponseObject]{
				Success: resp,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
	})
}

func withStoredIssuers(t *testing.T, s storage.Engine, issuers ...*types.Issuer) {
	seedCtx, err := s.BeginContext(context.Background())
	if err != nil {
		assert.FailNow(t, "failed to begin context")
	}

	for _, issuer := range issuers {
		i, err := s.CreateIssuer(seedCtx, *issuer)

		if !assert.NoError(t, err) {
			assert.FailNow(t, "error initializing issuer")
		}

		*issuer = *i
	}

	if err := s.CommitContext(seedCtx); err != nil {
		assert.FailNow(t, "error committing seed issuers")
	}
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

func withStoredUsers(t *testing.T, s storage.Engine, users ...*types.UserInfo) {
	seedCtx, err := s.BeginContext(context.Background())
	if err != nil {
		assert.FailNow(t, "failed to begin context")
	}

	for _, u := range users {
		user := u
		*user, err = s.StoreUserInfo(seedCtx, *user)

		if !assert.NoError(t, err) {
			assert.FailNow(t, "error initializing user")
		}
	}

	if err := s.CommitContext(seedCtx); err != nil {
		assert.FailNow(t, "error committing seed users")
	}
}
