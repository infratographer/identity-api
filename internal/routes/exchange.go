package routes

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/metal-toolbox/auditevent"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/auditx"
	"go.infratographer.com/identity-api/internal/types"
)

type tokenHandler struct {
	logger   *zap.SugaredLogger
	provider fosite.OAuth2Provider
}

func getOutcomeFromError(err error) string {
	rfcErr := fosite.ErrorToRFC6749Error(err)

	if rfcErr == nil {
		return auditevent.OutcomeFailed
	}

	status := rfcErr.StatusCode()

	if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		return auditevent.OutcomeDenied
	}

	if status >= http.StatusInternalServerError {
		return auditevent.OutcomeFailed
	}

	return auditevent.OutcomeSucceeded
}

func setContextFromError(c echo.Context, err error) {
	outcome := getOutcomeFromError(err)

	auditx.SetOutcome(c, outcome)

	var tokErr types.ErrorInvalidTokenRequest

	if !errors.As(err, &tokErr) {
		return
	}

	auditx.SetSubject(c, tokErr.Subject)
}

// Handle processes the request for the token handler.
func (h *tokenHandler) Handle(c echo.Context) error {
	var session oauth2.JWTSession

	ctx := c.Request().Context()

	accessRequest, err := h.provider.NewAccessRequest(ctx, c.Request(), &session)

	if err != nil {
		setContextFromError(c, err)

		h.logger.Errorf("Error occurred in NewAccessRequest: %+v", err)
		h.provider.WriteAccessError(ctx, c.Response(), accessRequest, err)

		return nil
	}

	subject := map[string]string{
		"subject": session.JWTClaims.Subject,
	}

	auditx.SetSubject(c, subject)

	response, err := h.provider.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessResponse: %+v", err)
		h.provider.WriteAccessError(ctx, c.Response(), accessRequest, err)

		return nil
	}

	// All done, send the response.
	h.provider.WriteAccessResponse(ctx, c.Response(), accessRequest, response)

	return nil
}
