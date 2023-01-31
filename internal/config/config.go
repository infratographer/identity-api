// Package config provides the configuration for the server.
package config

import (
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"

	"go.infratographer.com/identity-manager-sts/internal/fositex"
	"go.infratographer.com/identity-manager-sts/internal/storage"
)

// Config is the configuration for the application.
var Config struct {
	Server  ginx.Config
	Logging loggingx.Config
	OAuth   fositex.Config
	OTel    otelx.Config
	Storage storage.Config
}
