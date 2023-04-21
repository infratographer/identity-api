package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"go.uber.org/zap"
)

type tokenHandler struct {
	logger   *zap.SugaredLogger
	provider fosite.OAuth2Provider
}

// Handle processes the request for the token handler.
func (h *tokenHandler) Handle(c echo.Context) error {
	var session oauth2.JWTSession

	ctx := c.Request().Context()

	accessRequest, err := h.provider.NewAccessRequest(ctx, c.Request(), &session)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessRequest: %+v", err)
		h.provider.WriteAccessError(ctx, c.Response().Writer, accessRequest, err)

		return nil
	}

	response, err := h.provider.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessResponse: %+v", err)
		h.provider.WriteAccessError(ctx, c.Response().Writer, accessRequest, err)

		return nil
	}

	// All done, send the response.
	h.provider.WriteAccessResponse(ctx, c.Response().Writer, accessRequest, response)

	return nil
}
