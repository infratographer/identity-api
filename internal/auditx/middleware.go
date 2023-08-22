// Package auditx provides functions and data for auditing OAuth 2.0-specific events.
package auditx

import (
	"context"
	"io"

	"github.com/labstack/echo/v4"
	audithelpers "github.com/metal-toolbox/auditevent/helpers"
	"github.com/metal-toolbox/auditevent/middleware/echoaudit"
)

const (
	ctxKeyOutcome = "auditx.outcome"
	ctxKeySubject = "auditx.subject"
)

// SetSubject sets the subject of the auditable event.
func SetSubject(c echo.Context, subject map[string]string) {
	c.Set(ctxKeySubject, subject)
}

func getSubject(c echo.Context) map[string]string {
	maybeSubject := c.Get(ctxKeySubject)

	if maybeSubject == nil {
		return nil
	}

	return maybeSubject.(map[string]string)
}

// SetOutcome sets the outcome of the auditable event.
func SetOutcome(c echo.Context, outcome string) {
	c.Set(ctxKeyOutcome, outcome)
}

func getOutcome(c echo.Context) string {
	outcome := c.Get(ctxKeyOutcome)

	if outcome == nil {
		return echoaudit.GetOutcomeDefault(c)
	}

	return outcome.(string)
}

func newNopMiddleware() (*echoaudit.Middleware, func() error, error) {
	mdw := echoaudit.NewJSONMiddleware("", io.Discard)

	closer := func() error {
		return nil
	}

	return mdw, closer, nil
}

func NewMiddleware(ctx context.Context, c Config) (*echoaudit.Middleware, func() error, error) {
	if !c.Enabled {
		return newNopMiddleware()
	}

	f, err := audithelpers.OpenAuditLogFileUntilSuccessWithContext(ctx, c.Path)
	if err != nil {
		return nil, nil, err
	}

	closer := func() error {
		return f.Close()
	}

	mdw := echoaudit.NewJSONMiddleware(c.Component, f).
		WithPrometheusMetrics().
		WithSubjectHandler(getSubject).
		WithOutcomeHandler(getOutcome)

	return mdw, closer, nil
}
