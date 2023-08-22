// Package config provides the configuration for the server.
package config

import (
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"

	"go.infratographer.com/identity-api/internal/auditx"
	"go.infratographer.com/identity-api/internal/fositex"
)

// Config is the configuration for the application.
var Config struct {
	Server  echox.Config
	Logging loggingx.Config
	OAuth   fositex.Config
	OTel    otelx.Config
	CRDB    crdbx.Config
	Audit   auditx.Config
}
