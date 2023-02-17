// Package cmd provides the root command for the application.
package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/loggingx"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/config"
)

var (
	appName = "identity-api"
	rootCmd = &cobra.Command{
		Use:   "identity-api",
		Short: "identity-api authorization server",
	}

	cfgFile string
	logger  *zap.SugaredLogger
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/identity-api/identity-api.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/identity-api")
		viper.SetConfigType("yaml")
		viper.SetConfigName("identity-api")
	}

	// Allow populating configuration from environment
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("idapi")
	viper.AutomaticEnv() // read in environment variables that match

	err := viper.ReadInConfig()

	logger = loggingx.InitLogger(appName, config.Config.Logging)

	if err == nil {
		logger.Infow("using config file",
			"file", viper.ConfigFileUsed(),
		)
	}

	err = viper.Unmarshal(&config.Config)
	if err != nil {
		logger.Fatalw("unable to decode app config", "error", err)
	}
}

// Execute executes the root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
