// Package userinfo contains the enpdoints for translating
// STS tokens to original IdP user info.
package userinfo

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/x/echojwtx"
	"go.infratographer.com/x/urnx"
)

// Handler provides the endpoint for /userinfo
type Handler struct {
	store types.UserInfoService
}

// NewHandler creates a UserInfo handler with the storage engine
func NewHandler(userInfoSvc types.UserInfoService) (*Handler, error) {
	return &Handler{
		store: userInfoSvc,
	}, nil
}

// Handle expects an authenticated request using a STS token and returns
// the stored userinfo if it exists.
func (h *Handler) handle(ctx echo.Context) error {
	fullSubject := echojwtx.Actor(ctx)

	urn, err := urnx.Parse(fullSubject)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid subject").SetInternal(err)
	}

	resourceID := urn.ResourceID.String()

	info, err := h.store.LookupUserInfoByID(ctx.Request().Context(), resourceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	return ctx.JSON(http.StatusOK, info)
}

// Routes registers the userinfo handler in a echo.Group
func (h *Handler) Routes(rg *echo.Group) {
	rg.GET("userinfo", h.handle)
}
