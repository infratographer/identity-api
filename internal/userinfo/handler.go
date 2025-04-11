// Package userinfo contains the enpdoints for translating
// STS tokens to original IdP user info.
package userinfo

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"go.infratographer.com/x/echojwtx"
	"go.infratographer.com/x/gidx"

	"go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/types"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
)

// Store is an interface providing userinfo and group services
type Store interface {
	types.UserInfoService
	types.GroupService
}

// Handler provides the endpoint for /userinfo
type Handler struct {
	store Store
}

// NewHandler creates a UserInfo handler with the storage engine
func NewHandler(userInfoSvc Store) (*Handler, error) {
	return &Handler{
		store: userInfoSvc,
	}, nil
}

// Handle expects an authenticated request using a STS token and returns
// the stored userinfo if it exists.
func (h *Handler) handle(ctx echo.Context) error {
	fullSubject := echojwtx.Actor(ctx)

	resourceID, err := gidx.Parse(fullSubject)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid subject").SetInternal(err)
	}

	info, err := h.store.LookupUserInfoByID(ctx.Request().Context(), resourceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	return ctx.JSON(http.StatusOK, info)
}

// ListUserGroups expects an authenticated request using a STS token and
// returns the groups the user is a member of.
func (h *Handler) listUserGroups(ctx echo.Context) error {
	fullSubject := echojwtx.Actor(ctx)

	resourceID, err := gidx.Parse(fullSubject)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid subject").SetInternal(err)
	}

	cursor := ctx.QueryParam("cursor")
	limit := ctx.QueryParam("limit")

	pagination := v1.ListUserGroupsParams{}

	if cursor != "" {
		c := crdbx.Cursor(cursor)
		pagination.Cursor = &c
	}

	if limit != "" {
		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid limit: %s", limit))
		}

		l := crdbx.Limit(limitInt)
		pagination.Limit = &l
	}

	groups, err := h.store.ListGroupsBySubject(ctx.Request().Context(), resourceID, pagination)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	resp := groups.ToPrefixedIDs()

	collection := v1.GroupIDCollection{
		GroupIDs:   resp,
		Pagination: v1.Pagination{},
	}

	if err := pagination.SetPagination(&collection); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return ctx.JSON(http.StatusOK, collection)
}

// Routes registers the userinfo handler in a echo.Group
func (h *Handler) Routes(rg *echo.Group) {
	rg.GET("userinfo", h.handle)
	rg.GET("userinfo/groups", h.listUserGroups)
}
