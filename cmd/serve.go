package cmd

import (
	"context"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/dmv/internal/config"
	"go.infratographer.com/dmv/pkg/fositex"
	"go.infratographer.com/dmv/pkg/jwks"
	"go.infratographer.com/dmv/pkg/rfc8693"
	"go.infratographer.com/dmv/pkg/routes"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/otelx"
	"go.infratographer.com/x/versionx"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts DMV",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

var (
	defaultListen = ":8080"
)

func init() {
	rootCmd.AddCommand(serveCmd)

	ginx.MustViperFlags(viper.GetViper(), serveCmd.Flags(), defaultListen)
	otelx.MustViperFlags(viper.GetViper(), serveCmd.Flags())
}

func serve(ctx context.Context) {
	err := otelx.InitTracer(config.Config.OTel, appName, logger)
	if err != nil {
		logger.Fatalf("error initializing tracing: %s", err)
	}

	jwksStrategy := jwks.NewIssuerJWKSURIStrategy(config.Config.OAuth.SubjectTokenIssuers)
	oauth2Config, err := fositex.NewOAuth2Config(config.Config.OAuth)
	if err != nil {
		logger.Fatalf("error loading config: %s", err)
	}

	oauth2Config.IssuerJWKSURIStrategy = jwksStrategy

	keyGetter := func(ctx context.Context) (any, error) {
		return oauth2Config.GetSigningKey(ctx), nil
	}

	hmacStrategy := compose.NewOAuth2HMACStrategy(oauth2Config)
	jwtStrategy := compose.NewOAuth2JWTStrategy(keyGetter, hmacStrategy, oauth2Config)
	store := storage.NewExampleStore()
	tokenExchangeHandler := rfc8693.NewTokenExchangeHandler(oauth2Config, jwtStrategy, store)
	oauth2Config.TokenEndpointHandlers.Append(tokenExchangeHandler)
	provider := fositex.NewOAuth2Provider(oauth2Config, store)

	router := routes.NewRouter(logger, oauth2Config, provider)

	server := ginx.NewServer(logger.Desugar(), config.Config.Server, versionx.BuildDetails())
	server = server.AddHandler(router)

	server.Run()
}
