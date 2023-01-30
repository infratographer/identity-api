// Package cmd provides the root command for the application.
package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/loggingx"
	"go.uber.org/zap"

	"go.infratographer.com/identity-manager-sts/internal/config"
)

var (
	appName = "identity-manager-sts"
	rootCmd = &cobra.Command{
		Use:   "identity-manager-sts",
		Short: "identity-manager-sts authorization server",
	}

	cfgFile string
	logger  *zap.SugaredLogger
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/identity-manager-sts/identity-manager-sts.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/identity-manager-sts")
		viper.SetConfigType("yaml")
		viper.SetConfigName("identity-manager-sts")
	}

	// Allow populating configuration from environment
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("imsts")
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
