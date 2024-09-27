package httpsrv

import (
	"context"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	pagination "go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/gidx"
)

func TestGroupMembersAPIHandler(t *testing.T) {
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

	ownerID := gidx.MustNewID("testten")

	config := crdbx.Config{
		URI: testServer.PGURL().String(),
	}

	store, err := storage.NewEngine(config, storage.WithMigrations())
	if !assert.NoError(t, err) {
		assert.FailNow(t, "initialization failed")
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

	t.Run("ListGroupMembers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{engine: store}

		testGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-list-group-members",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroup, someMembers...)

		tc := []testingx.TestCase[ListGroupMembersRequestObject, ListGroupMembersResponseObject]{
			{
				Name:      "Invalid group id",
				Input:     ListGroupMembersRequestObject{GroupID: "definitely not a valid group id"},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupMembersResponseObject]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name:      "Group not found",
				Input:     ListGroupMembersRequestObject{GroupID: gidx.MustNewID(types.IdentityGroupIDPrefix)},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupMembersResponseObject]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name:      "Success default pagination",
				Input:     ListGroupMembersRequestObject{GroupID: testGroup.ID},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupMembersResponseObject]) {
					assert.Nil(t, res.Err)
					assert.IsType(t, ListGroupMembers200JSONResponse{}, res.Success)

					members := res.Success.(ListGroupMembers200JSONResponse)
					assert.Len(t, members.MemberIDs, len(someMembers))
					assert.NotNil(t, members.Pagination.Limit)
				},
			},
			{
				Name: "Success custom pagination",
				Input: ListGroupMembersRequestObject{
					GroupID: testGroup.ID,
					Params: v1.ListGroupMembersParams{
						Limit: ptr(1),
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupMembersResponseObject]) {
					assert.Nil(t, res.Err)
					assert.IsType(t, ListGroupMembers200JSONResponse{}, res.Success)

					members := res.Success.(ListGroupMembers200JSONResponse)
					assert.Len(t, members.MemberIDs, 1)
					assert.Equal(t, members.Pagination.Limit, 1)
					assert.NotNil(t, members.Pagination.Next)
				},
			},
		}

		runFn := func(ctx context.Context, input ListGroupMembersRequestObject) testingx.TestResult[ListGroupMembersResponseObject] {
			ctx = pagination.AsOfSystemTime(ctx, "")
			resp, err := handler.ListGroupMembers(ctx, input)

			return testingx.TestResult[ListGroupMembersResponseObject]{Success: resp, Err: err}
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("AddGroupMembers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{engine: store}

		testGroupWithNoMembers := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-add-group-members",
		}

		testGroupWithSomeMembers := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-add-group-members-with-some-members",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroupWithNoMembers)
		withStoredGroupAndMembers(t, store, testGroupWithSomeMembers, someMembers...)

		tc := []testingx.TestCase[AddGroupMembersRequestObject, []gidx.PrefixedID]{
			{
				Name:      "Invalid group id",
				Input:     AddGroupMembersRequestObject{GroupID: "definitely not a valid group id"},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Group not found",
				Input: AddGroupMembersRequestObject{
					GroupID: gidx.MustNewID(types.IdentityGroupIDPrefix),
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Invalid member id",
				Input: AddGroupMembersRequestObject{
					GroupID: testGroupWithNoMembers.ID,
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{"definitely not a valid member id"},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Success",
				Input: AddGroupMembersRequestObject{
					GroupID: testGroupWithNoMembers.ID,
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, 1)
				},
			},
			{
				Name: "Success with adding existing members",
				Input: AddGroupMembersRequestObject{
					GroupID: testGroupWithSomeMembers.ID,
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{someMembers[0]},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, len(someMembers))
				},
			},
		}

		runFn := func(ctx context.Context, input AddGroupMembersRequestObject) testingx.TestResult[[]gidx.PrefixedID] {
			_, err := handler.AddGroupMembers(ctx, input)
			if err != nil {
				return testingx.TestResult[[]gidx.PrefixedID]{Err: err}
			}

			if err := store.CommitContext(ctx); err != nil {
				return testingx.TestResult[[]gidx.PrefixedID]{Err: err}
			}

			ctx = context.Background()
			ctx = pagination.AsOfSystemTime(ctx, "")
			p := v1.ListGroupMembersParams{}
			mm, err := store.ListGroupMembers(ctx, input.GroupID, p)

			return testingx.TestResult[[]gidx.PrefixedID]{Success: mm, Err: err}
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("RemoveGroupMember", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{engine: store}

		testGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-remove-group-member",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroup, someMembers...)

		tc := []testingx.TestCase[RemoveGroupMemberRequestObject, []gidx.PrefixedID]{
			{
				Name: "Invalid group id",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   "definitely not a valid group id",
					SubjectID: gidx.MustNewID(types.IdentityUserIDPrefix),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Invalid member id",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   testGroup.ID,
					SubjectID: "definitely not a valid member id",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Group not found",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   gidx.MustNewID(types.IdentityGroupIDPrefix),
					SubjectID: gidx.MustNewID(types.IdentityUserIDPrefix),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Member not found",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   testGroup.ID,
					SubjectID: gidx.MustNewID(types.IdentityUserIDPrefix),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Success",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   testGroup.ID,
					SubjectID: someMembers[0],
				},
				SetupFn:   setupFn,
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, len(someMembers)-1)
				},
			},
		}

		runFn := func(ctx context.Context, input RemoveGroupMemberRequestObject) testingx.TestResult[[]gidx.PrefixedID] {
			_, err := handler.RemoveGroupMember(ctx, input)
			if err != nil {
				return testingx.TestResult[[]gidx.PrefixedID]{Err: err}
			}

			if err := store.CommitContext(ctx); err != nil {
				return testingx.TestResult[[]gidx.PrefixedID]{Err: err}
			}

			ctx = context.Background()
			ctx = pagination.AsOfSystemTime(ctx, "")
			p := v1.ListGroupMembersParams{}
			mm, err := store.ListGroupMembers(ctx, input.GroupID, p)

			return testingx.TestResult[[]gidx.PrefixedID]{Success: mm, Err: err}
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("PutGroupMembers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{engine: store}

		testGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-put-group-members",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroup, someMembers...)

		tc := []testingx.TestCase[ReplaceGroupMembersRequestObject, []gidx.PrefixedID]{
			{
				Name: "Invalid group id",
				Input: ReplaceGroupMembersRequestObject{
					GroupID: "definitely not a valid group id",
					Body: &v1.ReplaceGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Invalid member id",
				Input: ReplaceGroupMembersRequestObject{
					GroupID: testGroup.ID,
					Body: &v1.ReplaceGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{"definitely not a valid member id"},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Group not found",
				Input: ReplaceGroupMembersRequestObject{
					GroupID: gidx.MustNewID(types.IdentityGroupIDPrefix),
					Body: &v1.ReplaceGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Success",
				Input: ReplaceGroupMembersRequestObject{
					GroupID: testGroup.ID,
					Body: &v1.ReplaceGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, 1)
				},
			},
		}

		runFn := func(ctx context.Context, input ReplaceGroupMembersRequestObject) testingx.TestResult[[]gidx.PrefixedID] {
			_, err := handler.ReplaceGroupMembers(ctx, input)
			if err != nil {
				return testingx.TestResult[[]gidx.PrefixedID]{Err: err}
			}

			if err := store.CommitContext(ctx); err != nil {
				return testingx.TestResult[[]gidx.PrefixedID]{Err: err}
			}

			ctx = context.Background()
			ctx = pagination.AsOfSystemTime(ctx, "")
			p := v1.ListGroupMembersParams{}
			mm, err := store.ListGroupMembers(ctx, input.GroupID, p)

			return testingx.TestResult[[]gidx.PrefixedID]{Success: mm, Err: err}
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})
}

func withStoredGroupAndMembers(t *testing.T, s storage.Engine, group *types.Group, m ...gidx.PrefixedID) {
	seedCtx, err := s.BeginContext(context.Background())
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to begin context")
	}

	g, err := s.CreateGroup(seedCtx, *group)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to create group")
	}

	*group = *g

	if err := s.AddGroupMembers(seedCtx, group.ID, m...); !assert.NoError(t, err) {
		assert.FailNow(t, "failed to add members")
	}

	if err := s.CommitContext(seedCtx); !assert.NoError(t, err) {
		assert.FailNow(t, "error committing seed groups")
	}
}
