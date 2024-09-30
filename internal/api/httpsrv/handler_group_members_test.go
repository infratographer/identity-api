package httpsrv

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	pagination "go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/events"
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

	events := events.NewEvents()

	beginTx := func(ctx context.Context) context.Context {
		tx, err := store.BeginContext(ctx)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "setup failed")
		}

		return tx
	}

	setupFn := func(ctx context.Context) context.Context {
		pub := testingx.NewTestPublisher()
		ctxp := pub.ContextWithPublisher(ctx)

		return beginTx(ctxp)
	}

	cleanupFn := func(ctx context.Context) {
		err := store.RollbackContext(ctx)
		assert.NoError(t, err)
	}

	t.Run("ListGroupMembers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: events,
		}

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

		handler := apiHandler{
			engine:       store,
			eventService: events,
		}

		testGroupWithNoMember := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-add-group-member",
		}

		theOtherTestGroupWithNoMember := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-add-group-member-1",
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

		withStoredGroupAndMembers(t, store, testGroupWithNoMember)
		withStoredGroupAndMembers(t, store, theOtherTestGroupWithNoMember)
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Invalid member id",
				Input: AddGroupMembersRequestObject{
					GroupID: testGroupWithNoMember.ID,
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{"definitely not a valid member id"},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Failed to publish event",
				Input: AddGroupMembersRequestObject{
					GroupID: theOtherTestGroupWithNoMember.ID,
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn: func(ctx context.Context) context.Context {
					pub := testingx.NewTestPublisher(testingx.TestPublisherWithError(fmt.Errorf("you bad bad"))) // nolint: goerr113
					ctxp := pub.ContextWithPublisher(ctx)

					return beginTx(ctxp)
				},
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Error(t, res.Err)
					assert.ErrorContains(t, res.Err, "failed to add group members in permissions API")

					// ensure no members were added
					mc, err := store.GroupMembersCount(context.Background(), theOtherTestGroupWithNoMember.ID)
					assert.NoError(t, err)
					assert.Equal(t, 0, mc)
				},
			},
			{
				Name: "Success",
				Input: AddGroupMembersRequestObject{
					GroupID: testGroupWithNoMember.ID,
					Body: &v1.AddGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn:   setupFn,
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, 1)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Len(t, pub.CalledWith(), 1)

					cw := pub.CalledWith()[0]
					assert.Equal(t, testingx.TestPublisherMethodCreate, cw.Method)
					assert.Len(t, cw.Relations, len(res.Success))
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, len(someMembers))

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Len(t, pub.CalledWith(), 1)

					cw := pub.CalledWith()[0]
					assert.Equal(t, testingx.TestPublisherMethodCreate, cw.Method)
					assert.Len(t, cw.Relations, 1)
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
			mm, err := store.ListGroupMembers(ctx, input.GroupID, nil)

			return testingx.TestResult[[]gidx.PrefixedID]{Success: mm, Err: err}
		}

		ctx := testingx.NewTestPublisher().ContextWithPublisher(ctxPermsAllow(context.Background()))
		testingx.RunTests(ctx, t, tc, runFn)
	})

	t.Run("RemoveGroupMember", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: events,
		}

		testGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-remove-group-member",
		}

		theOtherTestGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-remove-group-member-1",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroup, someMembers...)
		withStoredGroupAndMembers(t, store, theOtherTestGroup, someMembers...)

		tc := []testingx.TestCase[RemoveGroupMemberRequestObject, []gidx.PrefixedID]{
			{
				Name: "Invalid group id",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   "definitely not a valid group id",
					SubjectID: gidx.MustNewID(types.IdentityUserIDPrefix),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Failed to publish event",
				Input: RemoveGroupMemberRequestObject{
					GroupID:   theOtherTestGroup.ID,
					SubjectID: someMembers[0],
				},
				SetupFn: func(ctx context.Context) context.Context {
					pub := testingx.NewTestPublisher(testingx.TestPublisherWithError(fmt.Errorf("you bad bad"))) // nolint: goerr113
					ctxp := pub.ContextWithPublisher(ctx)

					return beginTx(ctxp)
				},
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Error(t, res.Err)

					// ensure the member is still in the group
					mc, err := store.GroupMembersCount(context.Background(), theOtherTestGroup.ID)
					assert.NoError(t, err)
					assert.Len(t, someMembers, mc)
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, len(someMembers)-1)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Len(t, pub.CalledWith(), 1)

					cw := pub.CalledWith()[0]
					assert.Equal(t, testingx.TestPublisherMethodDelete, cw.Method)
					assert.Equal(t, cw.ResourceID, testGroup.ID)
					assert.Len(t, cw.Relations, 1)
					assert.Equal(t, someMembers[0], cw.Relations[0].SubjectID)
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
			mm, err := store.ListGroupMembers(ctx, input.GroupID, nil)

			return testingx.TestResult[[]gidx.PrefixedID]{Success: mm, Err: err}
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("PutGroupMembers", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: events,
		}

		testGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-put-group-members",
		}

		theOtherTestGroup := &types.Group{
			ID:      gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID: ownerID,
			Name:    "test-put-group-members-1",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroup, someMembers...)
		withStoredGroupAndMembers(t, store, theOtherTestGroup, someMembers...)

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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Success)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Failed to publish event",
				Input: ReplaceGroupMembersRequestObject{
					GroupID: theOtherTestGroup.ID,
					Body: &v1.ReplaceGroupMembersJSONRequestBody{
						MemberIDs: []gidx.PrefixedID{gidx.MustNewID(types.IdentityUserIDPrefix)},
					},
				},
				SetupFn: func(ctx context.Context) context.Context {
					pub := testingx.NewTestPublisher(testingx.TestPublisherWithError(fmt.Errorf("you bad bad"))) // nolint: goerr113
					ctxp := pub.ContextWithPublisher(ctx)

					return beginTx(ctxp)
				},
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Error(t, res.Err)
					assert.ErrorContains(t, res.Err, "failed to replace group members in permissions API")

					// ensure no members were added
					mc, err := store.GroupMembersCount(context.Background(), theOtherTestGroup.ID)
					assert.NoError(t, err)
					assert.Equal(t, len(someMembers), mc)
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
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[[]gidx.PrefixedID]) {
					assert.Nil(t, res.Err)
					assert.Len(t, res.Success, 1)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Len(t, pub.CalledWith(), 2)

					for _, cw := range pub.CalledWith() {
						assert.Equal(t, testGroup.ID, cw.ResourceID)

						if cw.Method == testingx.TestPublisherMethodCreate {
							assert.Len(t, cw.Relations, 1)
						} else if cw.Method == testingx.TestPublisherMethodDelete {
							assert.Len(t, cw.Relations, len(someMembers))
						}
					}
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
			mm, err := store.ListGroupMembers(ctx, input.GroupID, nil)

			return testingx.TestResult[[]gidx.PrefixedID]{Success: mm, Err: err}
		}

		ctx := testingx.NewTestPublisher().ContextWithPublisher(ctxPermsAllow(context.Background()))
		testingx.RunTests(ctx, t, tc, runFn)
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
