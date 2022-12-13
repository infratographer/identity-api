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
func (h *tokenHandler) Handle(ctx *gin.Context) {
	var session oauth2.JWTSession

	accessRequest, err := h.provider.NewAccessRequest(ctx, ctx.Request, &session)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessRequest: %+v", err)
		h.provider.WriteAccessError(ctx, ctx.Writer, accessRequest, err)
		return
	}

	response, err := h.provider.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		h.logger.Errorf("Error occurred in NewAccessResponse: %+v", err)
		h.provider.WriteAccessError(ctx, ctx.Writer, accessRequest, err)
		return
	}

	// All done, send the response.
	h.provider.WriteAccessResponse(ctx, ctx.Writer, accessRequest, response)
}
