package config

import (
	"go.infratographer.com/dmv/pkg/fositex"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"
)

var Config struct {
	Server  ginx.Config
	Logging loggingx.Config
	OAuth   fositex.Config
	OTel    otelx.Config
}
