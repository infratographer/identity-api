package auditx

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/viperx"
)

// Config represents an audit middleware configuration.
type Config struct {
	Enabled   bool
	Path      string
	Component string
}

// MustViperFlags sets the flags needed for auditing to work.
func MustViperFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.Bool("audit-enabled", false, "enable auditing")
	viperx.MustBindFlag(v, "audit.enabled", flags.Lookup("audit-enabled"))
	flags.String("audit-path", "", "audit log path")
	viperx.MustBindFlag(v, "audit.path", flags.Lookup("audit-path"))
	flags.String("audit-component", "", "audit component")
	viperx.MustBindFlag(v, "audit.component", flags.Lookup("audit-component"))
}
