package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/loggingx"
	"go.uber.org/zap"

	"go.infratographer.com/dmv/internal/config"
)

var (
	appName = "dmv"
	rootCmd = &cobra.Command{
		Use:   "dmv",
		Short: "DMV authorization server",
	}

	cfgFile string
	logger  *zap.SugaredLogger
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/dmv/dmv.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/dmv")
		viper.SetConfigType("yaml")
		viper.SetConfigName("dmv")
	}

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

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
