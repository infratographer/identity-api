package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/storage"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "runs identity-api database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		migrate(cmd.Context())
	},
}

func migrate(ctx context.Context) {
	logger.Info("running database migrations")

	err := storage.RunMigrations(config.Config.Storage)
	if err != nil {
		logger.Fatalf("error running migrations: %s", err)
	}

	logger.Info("success")
}
