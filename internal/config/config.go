// Package config provides the configuration for the server.
package config

import (
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"

	"go.infratographer.com/dmv/pkg/fositex"
)

// Config is the configuration for the application.
var Config struct {
	Server  ginx.Config
	Logging loggingx.Config
	OAuth   fositex.Config
	OTel    otelx.Config
}
