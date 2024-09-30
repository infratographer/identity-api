package httpsrv

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pagination "go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/events"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/testingx"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/gidx"
)

func TestGroupAPIHandler(t *testing.T) {
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

	es := events.NewEvents()

	t.Run("GetGroup", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: es,
		}

		getGroupTestGroup := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     ownerID,
			Name:        "test-getgroup",
			Description: "it's a group for testing get group",
		}

		withStoredGroups(t, store, getGroupTestGroup)

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

		testCases := []testingx.TestCase[GetGroupByIDRequestObject, GetGroupByIDResponseObject]{
			{
				Name: "Invalid group id",
				Input: GetGroupByIDRequestObject{
					GroupID: "notavalidid",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetGroupByIDResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Group not found",
				Input: GetGroupByIDRequestObject{
					GroupID: gidx.MustNewID(types.IdentityGroupIDPrefix),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetGroupByIDResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Success",
				Input: GetGroupByIDRequestObject{
					GroupID: getGroupTestGroup.ID,
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[GetGroupByIDResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, GetGroupByID200JSONResponse{}, res.Success)
					item := v1.Group(res.Success.(GetGroupByID200JSONResponse))
					assert.Equal(t, getGroupTestGroup.ID, item.ID)
					assert.Equal(t, getGroupTestGroup.Name, item.Name)
					assert.Equal(t, getGroupTestGroup.Description, *item.Description)
				},
			},
		}

		runFn := func(ctx context.Context, input GetGroupByIDRequestObject) testingx.TestResult[GetGroupByIDResponseObject] {
			resp, err := handler.GetGroupByID(ctx, input)
			return testingx.TestResult[GetGroupByIDResponseObject]{Success: resp, Err: err}
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, testCases, runFn)
	})

	t.Run("CreateGroup", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: es,
		}

		createGroupTestGroup := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     ownerID,
			Name:        "test-creategroup",
			Description: "it's a group for testing create group",
		}

		withStoredGroups(t, store, createGroupTestGroup)

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

		runFn := func(ctx context.Context, input CreateGroupRequestObject) testingx.TestResult[CreateGroupResponseObject] {
			resp, err := handler.CreateGroup(ctx, input)

			return testingx.TestResult[CreateGroupResponseObject]{Success: resp, Err: err}
		}

		tc := []testingx.TestCase[CreateGroupRequestObject, CreateGroupResponseObject]{
			{
				Name: "Invalid owner ID",
				Input: CreateGroupRequestObject{
					OwnerID: "notavalidid",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[CreateGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "No group name provided",
				Input: CreateGroupRequestObject{
					OwnerID: ownerID,
					Body: &v1.CreateGroupJSONRequestBody{
						Name: "",
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[CreateGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Duplicate group name",
				Input: CreateGroupRequestObject{
					OwnerID: ownerID,
					Body: &v1.CreateGroupJSONRequestBody{
						Name: createGroupTestGroup.Name,
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[CreateGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err, "unexpected error type", res.Err.Error())
					assert.Equal(t, http.StatusConflict, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Fail to publish group relationship",
				Input: CreateGroupRequestObject{
					OwnerID: ownerID,
					Body: &v1.CreateGroupJSONRequestBody{
						Name:        "test-creategroup-1",
						Description: ptr("it's a group for testing create group"),
					},
				},
				CleanupFn: func(_ context.Context) {},
				SetupFn: func(ctx context.Context) context.Context {
					p := testingx.NewTestPublisher(testingx.TestPublisherWithError(fmt.Errorf("you bad bad"))) // nolint: goerr113
					ctxp := p.ContextWithPublisher(ctx)

					return beginTx(ctxp)
				},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[CreateGroupResponseObject]) {
					assert.NotNil(t, res.Err)
					assert.Nil(t, res.Success)

					// ensure group will not be created
					ctx := context.Background()
					ctx = pagination.AsOfSystemTime(ctx, "")
					p := v1.ListGroupsParams{}
					g, err := store.ListGroupsByOwner(ctx, ownerID, p)
					assert.NoError(t, err)

					for _, gg := range g {
						if gg.OwnerID == ownerID {
							assert.NotEqual(t, "test-creategroup-1", gg.Name)
						}
					}
				},
			},
			{
				Name: "Success",
				Input: CreateGroupRequestObject{
					OwnerID: ownerID,
					Body: &v1.CreateGroupJSONRequestBody{
						Name:        "test-creategroup-2",
						Description: ptr("new group description"),
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[CreateGroupResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, CreateGroup200JSONResponse{}, res.Success)
					item := v1.Group(res.Success.(CreateGroup200JSONResponse))
					assert.NotEmpty(t, item.ID)
					assert.Equal(t, "test-creategroup-2", item.Name)
					assert.Equal(t, "new group description", *item.Description)

					group, err := store.GetGroupByID(ctx, item.ID)
					require.NoError(t, err, "unexpected error fetching group")
					assert.Equal(t, item.ID, group.ID)
					assert.Equal(t, item.Name, group.Name)
					assert.Equal(t, *item.Description, group.Description)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Len(t, pub.CalledWith(), 1)

					cw := pub.CalledWith()[0]
					assert.Equal(t, testingx.TestPublisherMethodCreate, cw.Method)
					assert.Equal(t, events.GroupTopic, cw.Topic)
					assert.Equal(t, item.ID, cw.ResourceID)
					assert.Equal(t, *item.OwnerID, cw.Relations[0].SubjectID)
				},
			},
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("ListGroups", func(t *testing.T) {
		t.Parallel()

		const numOfGroups = 5

		listGroupsTestGroups := make([]*types.Group, numOfGroups)
		theOtherOwnerID := gidx.MustNewID("testten")
		handler := apiHandler{
			engine:       store,
			eventService: es,
		}

		for i := 0; i < numOfGroups; i++ {
			listGroupsTestGroups[i] = &types.Group{
				ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
				OwnerID:     theOtherOwnerID,
				Name:        fmt.Sprintf("test-listgroup-%d", i),
				Description: "it's a group for testing list groups",
			}
		}

		withStoredGroups(t, store, listGroupsTestGroups...)

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

		runFn := func(ctx context.Context, input ListGroupsRequestObject) testingx.TestResult[ListGroupsResponseObject] {
			ctx = pagination.AsOfSystemTime(ctx, "")
			resp, err := handler.ListGroups(ctx, input)

			return testingx.TestResult[ListGroupsResponseObject]{Success: resp, Err: err}
		}

		tc := []testingx.TestCase[ListGroupsRequestObject, ListGroupsResponseObject]{
			{
				Name: "Invalid owner ID",
				Input: ListGroupsRequestObject{
					OwnerID: "notavalidid",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupsResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "No groups found",
				Input: ListGroupsRequestObject{
					OwnerID: gidx.MustNewID("tnntten"),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupsResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, ListGroups200JSONResponse{}, res.Success)
					items := res.Success.(ListGroups200JSONResponse)
					assert.Empty(t, items.Groups)
				},
			},
			{
				Name: "Success default pagination",
				Input: ListGroupsRequestObject{
					OwnerID: theOtherOwnerID,
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupsResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, ListGroups200JSONResponse{}, res.Success)
					items := res.Success.(ListGroups200JSONResponse)
					assert.Len(t, items.Groups, numOfGroups)
					assert.Equal(t, items.Pagination.Limit, 10)
				},
			},
			{
				Name: "Success custom pagination",
				Input: ListGroupsRequestObject{
					OwnerID: theOtherOwnerID,
					Params: v1.ListGroupsParams{
						Limit: ptr(2),
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[ListGroupsResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, ListGroups200JSONResponse{}, res.Success)
					items := res.Success.(ListGroups200JSONResponse)
					assert.Len(t, items.Groups, 2)
					assert.Equal(t, items.Pagination.Limit, 2)
					assert.NotNil(t, items.Pagination.Next)
				},
			},
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("UpdateGroup", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: es,
		}

		theOtherOwnerID := gidx.MustNewID("testten")

		updateGroupTestGroup := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     ownerID,
			Name:        "test-updategroup",
			Description: "it's a group for testing update group",
		}

		theOtherGroup := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     ownerID,
			Name:        "test-updategroup-2",
			Description: "it's a group for testing update group",
		}

		theOtherOwnersGroup := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     theOtherOwnerID,
			Name:        "test-updategroup-3",
			Description: "it's a group for testing update group",
		}

		withStoredGroups(t, store, updateGroupTestGroup, theOtherGroup, theOtherOwnersGroup)

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

		runFn := func(ctx context.Context, input UpdateGroupRequestObject) testingx.TestResult[UpdateGroupResponseObject] {
			ctx = pagination.AsOfSystemTime(ctx, "")
			resp, err := handler.UpdateGroup(ctx, input)

			return testingx.TestResult[UpdateGroupResponseObject]{Success: resp, Err: err}
		}

		tc := []testingx.TestCase[UpdateGroupRequestObject, UpdateGroupResponseObject]{
			{
				Name: "Invalid group ID",
				Input: UpdateGroupRequestObject{
					GroupID: "notavalidid",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[UpdateGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Group not found",
				Input: UpdateGroupRequestObject{
					GroupID: gidx.MustNewID(types.IdentityGroupIDPrefix),
					Body:    &v1.UpdateGroupJSONRequestBody{},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[UpdateGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Conflicting group name",
				Input: UpdateGroupRequestObject{
					GroupID: theOtherGroup.ID,
					Body: &v1.UpdateGroupJSONRequestBody{
						Name: &updateGroupTestGroup.Name,
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[UpdateGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusConflict, res.Err.(*echo.HTTPError).Code)
				},
			},
			{
				Name: "Same name different owner",
				Input: UpdateGroupRequestObject{
					GroupID: theOtherOwnersGroup.ID,
					Body: &v1.UpdateGroupJSONRequestBody{
						Name: &updateGroupTestGroup.Name,
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[UpdateGroupResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, UpdateGroup200JSONResponse{}, res.Success)

					resp := res.Success.(UpdateGroup200JSONResponse)

					assert.Equal(t, updateGroupTestGroup.Name, resp.Name)
				},
			},
			{
				Name: "Success",
				Input: UpdateGroupRequestObject{
					GroupID: updateGroupTestGroup.ID,
					Body: &v1.UpdateGroupJSONRequestBody{
						Name:        ptr("test-updategroup-1234567"),
						Description: ptr("new description"),
					},
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[UpdateGroupResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, UpdateGroup200JSONResponse{}, res.Success)
					resp := res.Success.(UpdateGroup200JSONResponse)
					assert.Equal(t, "test-updategroup-1234567", resp.Name)
					assert.Equal(t, "new description", *resp.Description)

					group, err := store.GetGroupByID(ctx, updateGroupTestGroup.ID)
					require.NoError(t, err, "unexpected error fetching group")
					assert.Equal(t, resp.Name, group.Name)
					assert.Equal(t, *resp.Description, group.Description)
				},
			},
		}

		testingx.RunTests(ctxPermsAllow(context.Background()), t, tc, runFn)
	})

	t.Run("DeleteGroup", func(t *testing.T) {
		t.Parallel()

		handler := apiHandler{
			engine:       store,
			eventService: es,
		}

		deleteGroupTestGroup := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     ownerID,
			Name:        "test-deletegroup",
			Description: "it's a group for testing delete group",
		}

		withStoredGroups(t, store, deleteGroupTestGroup)

		testGroupWithSomeMembers := &types.Group{
			ID:          gidx.MustNewID(types.IdentityGroupIDPrefix),
			OwnerID:     ownerID,
			Name:        "test-deletegroup-2",
			Description: "it's a group for testing delete group",
		}

		someMembers := []gidx.PrefixedID{
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
			gidx.MustNewID(types.IdentityUserIDPrefix),
		}

		withStoredGroupAndMembers(t, store, testGroupWithSomeMembers, someMembers...)

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

		runFn := func(ctx context.Context, input DeleteGroupRequestObject) testingx.TestResult[DeleteGroupResponseObject] {
			resp, err := handler.DeleteGroup(ctx, input)
			return testingx.TestResult[DeleteGroupResponseObject]{Success: resp, Err: err}
		}

		tc := []testingx.TestCase[DeleteGroupRequestObject, DeleteGroupResponseObject]{
			{
				Name: "Invalid group ID",
				Input: DeleteGroupRequestObject{
					GroupID: "notavalidid",
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[DeleteGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Group not found",
				Input: DeleteGroupRequestObject{
					GroupID: gidx.MustNewID(types.IdentityGroupIDPrefix),
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[DeleteGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusNotFound, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Group with members",
				Input: DeleteGroupRequestObject{
					GroupID: testGroupWithSomeMembers.ID,
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[DeleteGroupResponseObject]) {
					assert.Error(t, res.Err)
					assert.IsType(t, &echo.HTTPError{}, res.Err)
					assert.Equal(t, http.StatusBadRequest, res.Err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Empty(t, pub.CalledWith())
				},
			},
			{
				Name: "Fail to publish group relationship",
				Input: DeleteGroupRequestObject{
					GroupID: deleteGroupTestGroup.ID,
				},
				SetupFn: func(ctx context.Context) context.Context {
					p := testingx.NewTestPublisher(testingx.TestPublisherWithError(fmt.Errorf("you bad bad"))) // nolint: goerr113
					ctxp := p.ContextWithPublisher(ctx)

					return beginTx(ctxp)
				},
				CleanupFn: func(_ context.Context) {},
				CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[DeleteGroupResponseObject]) {
					assert.NotNil(t, res.Err)
					assert.Nil(t, res.Success)

					// ensure group will not be deleted
					group, err := store.GetGroupByID(context.Background(), deleteGroupTestGroup.ID)
					assert.NoError(t, err)
					assert.NotNil(t, group)
				},
			},
			{
				Name: "Success",
				Input: DeleteGroupRequestObject{
					GroupID: deleteGroupTestGroup.ID,
				},
				SetupFn:   setupFn,
				CleanupFn: cleanupFn,
				CheckFn: func(ctx context.Context, t *testing.T, res testingx.TestResult[DeleteGroupResponseObject]) {
					assert.NoError(t, res.Err)
					assert.IsType(t, DeleteGroup200JSONResponse{}, res.Success)

					_, err := handler.GetGroupByID(ctx, GetGroupByIDRequestObject{GroupID: deleteGroupTestGroup.ID})
					assert.Error(t, err)
					assert.IsType(t, &echo.HTTPError{}, err)
					assert.Equal(t, http.StatusNotFound, err.(*echo.HTTPError).Code)

					pub, ok := testingx.GetPublisherFromContext(ctx)
					assert.Equal(t, true, ok)
					assert.Len(t, pub.CalledWith(), 1)

					cw := pub.CalledWith()[0]
					assert.Equal(t, testingx.TestPublisherMethodDelete, cw.Method)
					assert.Equal(t, events.GroupTopic, cw.Topic)
					assert.Equal(t, deleteGroupTestGroup.ID, cw.ResourceID)
				},
			},
		}

		ctx := testingx.NewTestPublisher().ContextWithPublisher(ctxPermsAllow(context.Background()))
		testingx.RunTests(ctx, t, tc, runFn)
	})
}

func withStoredGroups(t *testing.T, s storage.Engine, groups ...*types.Group) {
	seedCtx, err := s.BeginContext(context.Background())
	if err != nil {
		assert.FailNow(t, "failed to begin context")
	}

	for _, group := range groups {
		g, err := s.CreateGroup(seedCtx, *group)

		if !assert.NoError(t, err, "error initializing group %s: %v", group.Name, err) {
			assert.FailNow(t, "error initializing group", err, group.Name)
		}

		*group = *g
	}

	if err := s.CommitContext(seedCtx); err != nil {
		assert.FailNow(t, "error committing seed groups")
	}
}
