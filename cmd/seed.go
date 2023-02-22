package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/storage"
)

func init() {
	rootCmd.AddCommand(seedDatabaseCmd)
}

var seedDatabaseCmd = &cobra.Command{
	Use:   "seed-database",
	Short: "seeds identity-api database",
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
