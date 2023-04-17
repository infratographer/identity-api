package cmd

import (
	"context"

	"github.com/ory/fosite/compose"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/otelx"
	"go.infratographer.com/x/versionx"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/api/httpsrv"
	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/jwks"
	"go.infratographer.com/identity-api/internal/oauth2"
	"go.infratographer.com/identity-api/internal/rfc8693"
	"go.infratographer.com/identity-api/internal/routes"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/userinfo"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts identity-api",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

var (
	defaultListen = ":8080"
)

func init() {
	rootCmd.AddCommand(serveCmd)

	v := viper.GetViper()
	flags := serveCmd.Flags()

	crdbx.MustViperFlags(v, flags)
	echox.MustViperFlags(v, flags, defaultListen)
	otelx.MustViperFlags(v, flags)
}

func serve(_ context.Context) {
	err := otelx.InitTracer(config.Config.OTel, appName, logger)
	if err != nil {
		logger.Fatalf("error initializing tracing: %s", err)
	}

	var engineOpts []storage.EngineOption

	if config.Config.OTel.Enabled {
		engineOpts = append(engineOpts, storage.WithTracing(config.Config.CRDB))
	}

	storageEngine, err := storage.NewEngine(config.Config.CRDB, engineOpts...)
	if err != nil {
		logger.Fatalf("error initializing storage: %s", err)
	}

	mappingStrategy := rfc8693.NewClaimMappingStrategy(storageEngine)

	issuerJWKSURIProvider := jwks.NewIssuerJWKSURIProvider(storageEngine)

	oauth2Config, err := fositex.NewOAuth2Config(config.Config.OAuth)
	if err != nil {
		logger.Fatalf("error loading config: %s", err)
	}

	oauth2Config.IssuerJWKSURIProvider = issuerJWKSURIProvider
	oauth2Config.ClaimMappingStrategy = mappingStrategy
	oauth2Config.UserInfoStrategy = storageEngine

	keyGetter := func(ctx context.Context) (any, error) {
		return oauth2Config.GetSigningKey(ctx), nil
	}

	hmacStrategy := compose.NewOAuth2HMACStrategy(oauth2Config)
	jwtStrategy := compose.NewOAuth2JWTStrategy(keyGetter, hmacStrategy, oauth2Config)

	provider := fositex.NewOAuth2Provider(
		oauth2Config,
		storageEngine,
		jwtStrategy,
		rfc8693.NewTokenExchangeHandler,
		oauth2.NewClientCredentialsHandlerFactory,
	)

	apiHandler, err := httpsrv.NewAPIHandler(storageEngine)
	if err != nil {
		logger.Fatal("error initializing API server: %s", err)
	}

	userInfoHandler, err := userinfo.NewHandler(storageEngine, oauth2Config)
	if err != nil {
		logger.Fatal("error initializing UserInfo handler: %s", err)
	}

	router := routes.NewRouter(logger, oauth2Config, provider, config.Config.OAuth.Issuer)

	srv := echox.NewServer(
		logger.Desugar(),
		echox.Config{
			Listen:              viper.GetString("server.listen"),
			ShutdownGracePeriod: viper.GetDuration("server.shutdown-grace-period"),
		},
		versionx.BuildDetails(),
	)

	srv.AddHandler(router)
	srv.AddHandler(apiHandler)
	srv.AddHandler(userInfoHandler)

	if err := srv.Run(); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}
