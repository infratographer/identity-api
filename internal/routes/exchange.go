package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"go.uber.org/zap"
)

type tokenHandler struct {
	logger   *zap.SugaredLogger
	provider fosite.OAuth2Provider
}

// Handle processes the request for the token handler.
func (h *tokenHandler) Handle(c *gin.Context) {
	var session oauth2.JWTSession

	ctx := c.Request.Context()

	accessRequest, err := h.provider.NewAccessRequest(ctx, c.Request, &session)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessRequest: %+v", err)
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)

		return
	}

	// set auditing fields from the session
	c.Set("jwt.subject", session.GetSubject())
	c.Set("jwt.user", session.GetUsername())

	response, err := h.provider.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessResponse: %+v", err)
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)

		return
	}

	// TODO infratographer specific subject and issuer info
	c.Set("audit.data", map[string]interface{}{
		"subject.issuer":         "TODO",
		"infratographer.subject": "TODO",
	})

	// All done, send the response.
	h.provider.WriteAccessResponse(ctx, c.Writer, accessRequest, response)
}
