package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/viperx"

	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/storage"
)

func init() {
	rootCmd.AddCommand(seedDatabaseCmd)

	v := viper.GetViper()
	flags := seedDatabaseCmd.Flags()

	crdbx.MustViperFlags(v, flags)

	flags.String("data", "", "location of data file on disk")
	viperx.MustBindFlag(v, "data", flags.Lookup("data"))
}

var seedDatabaseCmd = &cobra.Command{
	Use:   "seed-database",
	Short: "seeds identity-api database with seed data from storage configs",
	Run: func(cmd *cobra.Command, args []string) {
		seedDatabase(cmd.Context())
	},
}

func seedDatabase(_ context.Context) {
	logger.Info("seeding database")

	path := viper.GetString("data")
	if path == "" {
		logger.Fatal("no data path provided")
	}

	err := storage.SeedDatabase(config.Config.CRDB, path)
	if err != nil {
		logger.Fatalf("error seeding database: %s", err)
	}

	logger.Info("success")
}
