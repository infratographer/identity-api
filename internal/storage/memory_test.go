package storage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
)

func compareIssuers(t *testing.T, exp types.Issuer, obs types.Issuer) {
	// Clear out mappings, since some of the AST can change between marshaling/unmarshaling
	expMappings, err := exp.ClaimMappings.Repr()
	if err != nil {
		t.Fatal(err)
	}

	obsMappings, err := obs.ClaimMappings.Repr()
	if err != nil {
		t.Fatal(err)
	}

	exp.ClaimMappings = nil
	obs.ClaimMappings = nil

	assert.Equal(t, exp, obs)
	assert.Equal(t, expMappings, obsMappings)
}

func newDB(t *testing.T) (*sql.DB, func()) {
	db, shutdown := testserver.NewDBForTest(t)

	err := RunMigrations(db)
	if err != nil {
		shutdown()
		t.Fatal(err)
	}

	return db, shutdown
}

func TestIssuerService(t *testing.T) {
	t.Parallel()

	db, shutdown := newDB(t)

	t.Cleanup(func() {
		shutdown()
	})

	mappingStrs := map[string]string{
		"foo": "123",
	}

	mappings, err := types.NewClaimsMapping(mappingStrs)
	if err != nil {
		panic(err)
	}

	tenantID := "56a95c1b-33f8-4def-8b6d-ca9fe6976170"
	issuer := types.Issuer{
		TenantID:      tenantID,
		ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
		Name:          "Example",
		URI:           "https://example.com/",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: mappings,
	}

	config := Config{
		db: db,
		SeedData: SeedData{
			Issuers: []SeedIssuer{
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

	issSvc, err := newIssuerService(config)
	assert.Nil(t, err)

	t.Run("CreateIssuer", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            "6b0117f8-29e4-49fa-841e-63c52aa27d96",
			Name:          "Good issuer",
			URI:           "https://issuer-a27d96.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		testCases := []testingx.TestCase[types.Issuer, *types.Issuer]{
			{
				Name:  "Success",
				Input: issuer,
				SetupFn: func(ctx context.Context) context.Context {
					txCtx, err := beginTxContext(ctx, db)
					if !assert.NoError(t, err) {
						assert.FailNow(t, "setup failed")
					}

					return txCtx
				},
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
				CleanupFn: func(ctx context.Context) {
					err := rollbackContextTx(ctx)
					assert.NoError(t, err)
				},
			},
		}

		runFn := func(ctx context.Context, input types.Issuer) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.CreateIssuer(ctx, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetIssuerByURI", func(t *testing.T) {
		t.Parallel()

		testCases := []testingx.TestCase[string, *types.Issuer]{
			{
				Name:  "NotFound",
				Input: "https://evil.biz/",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
			{
				Name: "UsingTx",
				SetupFn: func(ctx context.Context) context.Context {
					txCtx, err := beginTxContext(ctx, db)
					if !assert.NoError(t, err) {
						assert.FailNow(t, "setup failed")
					}

					return txCtx
				},
				Input: "https://example.com/",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
				CleanupFn: func(ctx context.Context) {
					err := rollbackContextTx(ctx)
					assert.NoError(t, err)
				},
			},
			{
				Name:  "UsingDB",
				Input: "https://example.com/",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.GetIssuerByURI(ctx, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("GetIssuerByID", func(t *testing.T) {
		t.Parallel()

		testCases := []testingx.TestCase[string, *types.Issuer]{
			{
				Name:  "NotFound",
				Input: "00000000-0000-0000-0000-000000000000",
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
			},
			{
				Name:  "UsingTx",
				Input: issuer.ID,
				SetupFn: func(ctx context.Context) context.Context {
					txCtx, err := beginTxContext(ctx, db)
					if !assert.NoError(t, err) {
						assert.FailNow(t, "setup failed")
					}

					return txCtx
				},
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
				CleanupFn: func(ctx context.Context) {
					err := rollbackContextTx(ctx)
					assert.NoError(t, err)
				},
			},
			{
				Name:  "UsingDB",
				Input: issuer.ID,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					if assert.NoError(t, res.Err) {
						compareIssuers(t, issuer, *res.Success)
					}
				},
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.GetIssuerByID(ctx, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("UpdateIssuer", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            "b9ae2e16-11c0-49e4-8d9b-1d6698bba1a3",
			Name:          "Good issuer",
			URI:           "https://issuer-bba1a3.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		newName := "Better issuer"
		newURI := "https://issuer.info/better/"
		newJWKSURI := "https://issuer.info/better/jwks.json"
		newMapping := types.ClaimsMapping{}

		fullUpdate := types.IssuerUpdate{
			Name:          &newName,
			URI:           &newURI,
			JWKSURI:       &newJWKSURI,
			ClaimMappings: newMapping,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := beginTxContext(ctx, db)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = issSvc.CreateIssuer(ctx, issuer)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := rollbackContextTx(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[types.IssuerUpdate, *types.Issuer]{
			{
				Name:    "Full",
				Input:   fullUpdate,
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.Issuer]) {
					exp := issuer
					exp.Name = newName
					exp.URI = newURI
					exp.JWKSURI = newJWKSURI
					exp.ClaimMappings = newMapping

					if assert.NoError(t, res.Err) {
						compareIssuers(t, exp, *res.Success)
					}
				},
				CleanupFn: cleanupFn,
			},
		}

		runFn := func(ctx context.Context, input types.IssuerUpdate) testingx.TestResult[*types.Issuer] {
			iss, err := issSvc.UpdateIssuer(ctx, issuer.ID, input)

			result := testingx.TestResult[*types.Issuer]{
				Success: iss,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("DeleteIssuer", func(t *testing.T) {
		t.Parallel()

		issuer := types.Issuer{
			TenantID:      tenantID,
			ID:            "ace77968-03b0-4f1b-b3f0-b214daf4ac18",
			Name:          "Good issuer",
			URI:           "https://issuer-f4ac18.info/",
			JWKSURI:       "https://issuer.info/jwks.json",
			ClaimMappings: mappings,
		}

		setupFn := func(ctx context.Context) context.Context {
			ctx, err := beginTxContext(ctx, db)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			_, err = issSvc.CreateIssuer(ctx, issuer)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}

			return ctx
		}

		cleanupFn := func(ctx context.Context) {
			err := rollbackContextTx(ctx)
			assert.NoError(t, err)
		}

		testCases := []testingx.TestCase[string, any]{
			{
				Name:    "Success",
				Input:   issuer.ID,
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					if assert.NoError(t, res.Err) {
						_, err := issSvc.GetIssuerByID(ctx, issuer.ID)
						assert.ErrorIs(t, types.ErrorIssuerNotFound, err)
					}
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "NotFound",
				Input:   "00000000-0000-0000-0000-000000000000",
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[any]) {
					assert.ErrorIs(t, types.ErrorIssuerNotFound, res.Err)
				},
				CleanupFn: cleanupFn,
			},
		}

		runFn := func(ctx context.Context, input string) testingx.TestResult[any] {
			err = issSvc.DeleteIssuer(ctx, input)

			result := testingx.TestResult[any]{
				Success: nil,
				Err:     err,
			}

			return result
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})
}

func TestUserInfoStore(t *testing.T) {
	t.Parallel()

	db, shutdown := newDB(t)

	t.Cleanup(shutdown)

	tr := &recordingTransport{}
	httpClient := &http.Client{
		Transport: tr,
	}

	tenantID := "56a95c1b-33f8-4def-8b6d-ca9fe6976170"
	issuer := types.Issuer{
		TenantID:      tenantID,
		ID:            "e495a393-ae79-4a02-a78d-9798c7d9d252",
		Name:          "Example",
		URI:           "https://example.com/",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: types.ClaimsMapping{},
	}

	config := Config{
		db: db,
		SeedData: SeedData{
			Issuers: []SeedIssuer{
				{
					TenantID:      tenantID,
					ID:            issuer.ID,
					Name:          issuer.Name,
					URI:           issuer.URI,
					JWKSURI:       issuer.JWKSURI,
					ClaimMappings: map[string]string{},
				},
			},
		},
	}

	_, err := newIssuerService(config)
	assert.Nil(t, err)

	svc, err := newUserInfoService(config, WithHTTPClient(httpClient))
	assert.NoError(t, err)

	ctx := context.Background()
	user := types.UserInfo{
		Name:    "Maliketh",
		Email:   "mal@iketh.co",
		Issuer:  issuer.URI,
		Subject: "sub0|malikadmin",
	}

	var userInfoStored *types.UserInfo

	// seed the DB
	{
		ctx, err := beginTxContext(ctx, db)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "begin transaction for insert user failed")
		}

		userInfoStored, err = svc.StoreUserInfo(ctx, user)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "insert user failed")
		}

		err = commitContextTx(ctx)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "commit transaction insert user failed")
		}
	}

	if !assert.NoError(t, err) {
		assert.FailNow(t, "insert user failed")
	}

	setupFn := func(ctx context.Context) context.Context {
		ctx, err := beginTxContext(ctx, db)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "setup failed")
		}

		return ctx
	}

	cleanupFn := func(ctx context.Context) {
		err := rollbackContextTx(ctx)
		assert.NoError(t, err)
	}

	t.Run("LookupUserInfoByClaims", func(t *testing.T) {
		t.Parallel()

		type lookupType struct {
			issuer  string
			subject string
		}

		runFn := func(ctx context.Context, input lookupType) testingx.TestResult[*types.UserInfo] {
			out, err := svc.LookupUserInfoByClaims(ctx, input.issuer, input.subject)
			return testingx.TestResult[*types.UserInfo]{
				Success: out,
				Err:     err,
			}
		}

		testCases := []testingx.TestCase[lookupType, *types.UserInfo]{
			{
				Name:    "LoadAfterStore",
				Input:   lookupType{issuer: user.Issuer, subject: user.Subject},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, user, *res.Success)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "IncorrectIssuer",
				Input:   lookupType{issuer: user.Issuer + "foobar", subject: user.Subject},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrUserInfoNotFound)
					assert.Nil(t, res.Success)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "IncorrectSubject",
				Input:   lookupType{issuer: user.Issuer, subject: ""},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrUserInfoNotFound)
					assert.Nil(t, res.Success)
				},
				CleanupFn: cleanupFn,
			},
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("LookupUserInfoByID", func(t *testing.T) {
		t.Parallel()

		runFn := func(ctx context.Context, input string) (res testingx.TestResult[*types.UserInfo]) {
			out, err := svc.LookupUserInfoByID(ctx, input)
			res.Success = out
			res.Err = err

			return res
		}

		cases := []testingx.TestCase[string, *types.UserInfo]{
			{
				Name:    "Success",
				Input:   userInfoStored.ID.String(),
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, userInfoStored, res.Success)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "InvalidID",
				Input:   uuid.NewString(),
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrUserInfoNotFound)
				},
				CleanupFn: cleanupFn,
			},
		}

		testingx.RunTests(context.Background(), t, cases, runFn)
	})

	t.Run("FetchUserInfoFromIssuer", func(t *testing.T) {
		t.Parallel()

		type fetchInput struct {
			issuer   string
			token    string
			respBody *string
		}

		type fetchResult struct {
			tr recordingTransport
			ui *types.UserInfo
		}

		exampleResp := `
                  {
                    "name": "adam", "email": "ad@am.com",
                    "sub": "super-admin"
                  }`

		emptyResp := `{}`

		nullResp := `{"name": null, "email": null, "sub": null}`

		cases := []testingx.TestCase[fetchInput, fetchResult]{
			{
				Name:    "Success",
				Input:   fetchInput{issuer: "https://someidp.com", token: "supersecrettoken"},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[fetchResult]) {
					tr := res.Success.tr
					assert.Equal(t, "https://someidp.com/userinfo", tr.req.URL.String())
					assert.Equal(t, "Bearer supersecrettoken", tr.req.Header.Get("authorization"))
					assert.Equal(t, http.MethodGet, tr.req.Method)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "BadIssuer",
				Input:   fetchInput{issuer: "://", token: "supersecrettoken"},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[fetchResult]) {
					err := res.Err
					assert.ErrorContains(t, err, "missing protocol scheme")
				},
				CleanupFn: cleanupFn,
			},
			{
				Name: "FullFetch",
				Input: fetchInput{
					issuer:   "https://woo.com",
					token:    "supersecrettoken",
					respBody: &exampleResp,
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[fetchResult]) {
					info := res.Success.ui
					err := res.Err
					assert.NoError(t, err)
					expected := types.UserInfo{
						Name:    "adam",
						Email:   "ad@am.com",
						Subject: "super-admin",
						Issuer:  "https://woo.com",
					}

					assert.Equal(t, expected, *info)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name: "EmptyUserInfo",
				Input: fetchInput{
					issuer:   "https://woo.com",
					token:    "supersecretoken",
					respBody: &emptyResp,
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[fetchResult]) {
					err := res.Err
					assert.NoError(t, err)
					userinfo := res.Success.ui
					emptyUserInfo := types.UserInfo{
						ID:      uuid.Nil,
						Name:    "",
						Email:   "",
						Subject: "",
						Issuer:  "https://woo.com",
					}

					assert.Equal(t, emptyUserInfo, *userinfo)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name: "NullNameResponse",
				Input: fetchInput{
					issuer:   "https://woo.com",
					token:    "supersecretoken",
					respBody: &nullResp,
				},
				SetupFn: setupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[fetchResult]) {
					err := res.Err
					assert.NoError(t, err)
					userinfo := res.Success.ui
					emptyUserInfo := types.UserInfo{
						ID:      uuid.Nil,
						Name:    "",
						Email:   "",
						Subject: "",
						Issuer:  "https://woo.com",
					}

					assert.Equal(t, emptyUserInfo, *userinfo)
				},
				CleanupFn: cleanupFn,
			},
		}

		runFn := func(ctx context.Context, input fetchInput) testingx.TestResult[fetchResult] {
			tr := recordingTransport{body: input.respBody}
			client := http.Client{Transport: &tr}
			svc, err := newUserInfoService(config, WithHTTPClient(&client))
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create new fake transport: %v", err)
			}
			res, err := svc.FetchUserInfoFromIssuer(ctx, input.issuer, input.token)
			return testingx.TestResult[fetchResult]{
				Success: fetchResult{
					tr: tr,
					ui: res,
				},
				Err: err,
			}
		}
		testingx.RunTests(context.Background(), t, cases, runFn)
	})

	t.Run("StoreUserInfo", func(t *testing.T) {
		t.Parallel()

		caseSetupFn := func(ctx context.Context) context.Context {
			ctx = setupFn(ctx)
			ctx, err := beginTxContext(ctx, db)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "setup failed")
			}
			return ctx
		}

		caseCleanupFn := func(ctx context.Context) {
			err := rollbackContextTx(ctx)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to rollback after test case")
			}
		}

		cases := []testingx.TestCase[types.UserInfo, *types.UserInfo]{
			{
				Name: "UnboundIssuer",
				Input: types.UserInfo{
					Name:    "example-name",
					Email:   "example@example.web",
					Issuer:  "not-real",
					Subject: "user:person001",
				},
				SetupFn: caseSetupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrorIssuerNotFound)
				},
				CleanupFn: caseCleanupFn,
			},
			{
				Name: "Success",
				Input: types.UserInfo{
					Name:    "example-name",
					Email:   "example@example.web",
					Issuer:  "https://example.com/",
					Subject: "user:person001",
				},
				SetupFn: caseSetupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.NoError(t, res.Err)
				},
				CleanupFn: caseCleanupFn,
			},
			{
				Name: "EmptyIssuer",
				Input: types.UserInfo{
					Name:    "example-name",
					Email:   "example@example.web",
					Issuer:  "",
					Subject: "user:person001",
				},
				SetupFn: caseSetupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrInvalidUserInfo)
					assert.ErrorContains(t, res.Err, "issuer is empty")
				},
				CleanupFn: caseCleanupFn,
			},
			{
				Name:    "DuplicateEntryForSubject",
				Input:   user,
				SetupFn: caseSetupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, userInfoStored.ID, res.Success.ID)
				},
				CleanupFn: caseCleanupFn,
			},
			{
				Name: "EmptySubject",
				Input: types.UserInfo{
					Name:    "",
					Email:   "",
					Subject: "",
					Issuer:  "https://example.com",
				},
				SetupFn: caseSetupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[*types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrInvalidUserInfo)
					assert.ErrorContains(t, res.Err, "subject is empty")
				},
				CleanupFn: caseCleanupFn,
			},
		}

		runFn := func(ctx context.Context, input types.UserInfo) (out testingx.TestResult[*types.UserInfo]) {
			res, err := svc.StoreUserInfo(ctx, input)
			out.Success = res
			out.Err = err
			return out
		}
		testingx.RunTests(context.Background(), t, cases, runFn)
	})
}

type recordingTransport struct {
	req  *http.Request
	body *string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.req = req

	if rt.body != nil {
		resp := http.Response{
			Status:     http.StatusText(http.StatusOK),
			StatusCode: http.StatusOK,
		}

		r := io.NopCloser(bytes.NewReader([]byte(*rt.body)))
		resp.Body = r

		return &resp, nil
	}

	// Just error out to prevent making the network call, but we
	// can ensure the request is built properly
	return nil, errFakeHTTP
}

var errFakeHTTP = errors.New("error to stop http client from making a network call")
