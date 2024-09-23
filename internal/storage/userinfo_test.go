package storage

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.infratographer.com/identity-api/internal/celutils"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/x/gidx"
)

func TestUserInfoStore(t *testing.T) {
	t.Parallel()

	db, shutdown := testserver.NewDBForTest(t)

	err := runMigrations(db)
	if err != nil {
		shutdown()
		t.Fatal(err)
	}

	t.Cleanup(shutdown)

	ownerID := gidx.MustNewID("testten")
	issuer := types.Issuer{
		OwnerID:       ownerID,
		ID:            gidx.MustNewID("testiss"),
		Name:          "Example",
		URI:           "https://example.com",
		JWKSURI:       "https://example.com/.well-known/jwks.json",
		ClaimMappings: types.ClaimsMapping{},
	}

	celWithRemappedSubject, err := celutils.ParseCEL("claims.sub")
	assert.NoError(t, err)

	issuerRemappedSubject := types.Issuer{
		OwnerID: ownerID,
		ID:      gidx.MustNewID("testiss"),
		Name:    "Example Remapped Subject",
		URI:     "https://example2.com",
		JWKSURI: "https://example2.com/.well-known/jwks.json",
		ClaimMappings: types.ClaimsMapping{
			"prefixedid:sub": celWithRemappedSubject,
		},
	}

	issuerRemappedSubjectClaimRepr, err := issuerRemappedSubject.ClaimMappings.Repr()
	assert.NoError(t, err)

	seedIssuers := []SeedIssuer{
		{
			OwnerID:       ownerID,
			ID:            issuer.ID,
			Name:          issuer.Name,
			URI:           issuer.URI,
			JWKSURI:       issuer.JWKSURI,
			ClaimMappings: map[string]string{},
		},
		{
			OwnerID:       ownerID,
			ID:            issuerRemappedSubject.ID,
			Name:          issuerRemappedSubject.Name,
			URI:           issuerRemappedSubject.URI,
			JWKSURI:       issuerRemappedSubject.JWKSURI,
			ClaimMappings: issuerRemappedSubjectClaimRepr,
		},
	}

	issSvc, err := newIssuerService(db)
	assert.Nil(t, err)

	err = issSvc.seedDatabase(context.Background(), seedIssuers)
	assert.Nil(t, err)

	svc, err := newUserInfoService(db)
	assert.NoError(t, err)

	ctx := context.Background()
	user := types.UserInfo{
		Name:    "Maliketh",
		Email:   "mal@iketh.co",
		Issuer:  issuer.URI,
		Subject: "sub0|malikadmin",
	}

	userRemappedSub := types.UserInfo{
		Name:    "Ezekiel",
		Email:   "eze@kiel.co",
		Issuer:  issuerRemappedSubject.URI,
		Subject: "32cb2842-4a5d-45cc-b4f3-63cfcdf23e63",
	}

	// This user ID should be deterministically generated, so we precompute it here rather
	// than use generateSubjectID
	expUserInfoID, err := gidx.Parse("idntusr-JJ5-CXOzTNil-ncNcX8UIGzsDYSRGj1Ktc6oI-s9fSs")
	require.NoError(t, err)

	expUserInfoIDRemappedSub, err := gidx.Parse("idntusr-mqwvsgspnV9ZpGoefig_OV86qUt_3t9j5GLlF2EBVvc-32cb2842-4a5d-45cc-b4f3-63cfcdf23e63")
	require.NoError(t, err)

	// Ultimately we should test the evaluation of the claim mapping on token exchange, but
	// for the purposes of this we're just testing the storage of the user info based on
	// custom PrefixedIDs.
	userRemappedSub.ID = expUserInfoIDRemappedSub

	var userInfoStored types.UserInfo

	var userInfoRemappedSubStored types.UserInfo

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

		userInfoRemappedSubStored, err = svc.StoreUserInfo(ctx, userRemappedSub)
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

		runFn := func(ctx context.Context, input lookupType) testingx.TestResult[types.UserInfo] {
			out, err := svc.LookupUserInfoByClaims(ctx, input.issuer, input.subject)

			return testingx.TestResult[types.UserInfo]{
				Success: out,
				Err:     err,
			}
		}

		testCases := []testingx.TestCase[lookupType, types.UserInfo]{
			{
				Name:    "LoadAfterStore",
				Input:   lookupType{issuer: user.Issuer, subject: user.Subject},
				SetupFn: setupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, user, res.Success)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "IncorrectIssuer",
				Input:   lookupType{issuer: user.Issuer + "foobar", subject: user.Subject},
				SetupFn: setupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrUserInfoNotFound)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "IncorrectSubject",
				Input:   lookupType{issuer: user.Issuer, subject: ""},
				SetupFn: setupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrUserInfoNotFound)
				},
				CleanupFn: cleanupFn,
			},
		}

		testingx.RunTests(context.Background(), t, testCases, runFn)
	})

	t.Run("LookupUserInfoByID", func(t *testing.T) {
		t.Parallel()

		runFn := func(ctx context.Context, input gidx.PrefixedID) (res testingx.TestResult[types.UserInfo]) {
			out, err := svc.LookupUserInfoByID(ctx, input)
			res.Success = out
			res.Err = err

			return res
		}

		cases := []testingx.TestCase[gidx.PrefixedID, types.UserInfo]{
			{
				Name:    "Success",
				Input:   expUserInfoID,
				SetupFn: setupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, userInfoStored, res.Success)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "SuccessRemappedSubject",
				Input:   expUserInfoIDRemappedSub,
				SetupFn: setupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.NoError(t, res.Err)
					assert.Equal(t, userInfoRemappedSubStored, res.Success)
				},
				CleanupFn: cleanupFn,
			},
			{
				Name:    "InvalidID",
				Input:   gidx.MustNewID("invldid"),
				SetupFn: setupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.ErrorIs(t, res.Err, types.ErrUserInfoNotFound)
				},
				CleanupFn: cleanupFn,
			},
		}

		testingx.RunTests(context.Background(), t, cases, runFn)
	})

	t.Run("ParseUserInfoFromClaims", func(t *testing.T) {
		t.Parallel()

		cases := []testingx.TestCase[map[string]any, types.UserInfo]{
			{
				Name: "Success",
				Input: map[string]any{
					"iss":   "https://woo.com",
					"sub":   "badman@woo.com",
					"name":  "Badman",
					"email": "badman@woo.com",
				},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					exp := types.UserInfo{
						Issuer:  "https://woo.com",
						Subject: "badman@woo.com",
						Name:    "Badman",
						Email:   "badman@woo.com",
					}

					assert.NoError(t, res.Err)
					assert.Equal(t, exp, res.Success)
				},
			},
			{
				Name: "OnlySomeFields",
				Input: map[string]any{
					"iss": "https://woo.com",
					"sub": "badman@woo.com",
				},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					exp := types.UserInfo{
						Issuer:  "https://woo.com",
						Subject: "badman@woo.com",
						Name:    "",
						Email:   "",
					}

					assert.NoError(t, res.Err)
					assert.Equal(t, exp, res.Success)
				},
			},
			{
				Name: "MissingIssuer",
				Input: map[string]any{
					"sub": "badman@woo.com",
				},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.ErrorIs(t, errMissingClaim, res.Err)
				},
			},
			{
				Name: "MissingSubject",
				Input: map[string]any{
					"iss": "https://woo.com",
				},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.ErrorIs(t, errMissingClaim, res.Err)
				},
			},
			{
				Name: "MalformedClaim",
				Input: map[string]any{
					"iss": false,
				},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[types.UserInfo]) {
					assert.ErrorIs(t, errInvalidClaim, res.Err)
				},
			},
		}

		runFn := func(_ context.Context, input map[string]any) testingx.TestResult[types.UserInfo] {
			svc, err := newUserInfoService(db)
			require.NoError(t, err)

			userinfo, err := svc.ParseUserInfoFromClaims(input)

			out := testingx.TestResult[types.UserInfo]{
				Success: userinfo,
				Err:     err,
			}

			return out
		}

		testingx.RunTests(context.Background(), t, cases, runFn)
	})
}
