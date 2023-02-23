package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"

	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/storage"
)

func init() {
	rootCmd.AddCommand(seedDatabaseCmd)

	v := viper.GetViper()
	flags := seedDatabaseCmd.Flags()

	crdbx.MustViperFlags(v, flags)
}

var seedDatabaseCmd = &cobra.Command{
	Use:   "seed-database",
	Short: "seeds identity-api database with seed data from storage configs",
	Run: func(cmd *cobra.Command, args []string) {
		seedDatabase(cmd.Context())
	},
}

func seedDatabase(ctx context.Context) {
	logger.Info("seeding database")

	err := storage.SeedDatabase(config.Config.Storage)
	if err != nil {
		logger.Fatalf("error seeding database: %s", err)
	}

	logger.Info("success")
}
