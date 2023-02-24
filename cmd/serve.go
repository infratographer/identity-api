package cmd

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite/compose"
	fositestorage "github.com/ory/fosite/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/ginx"
	"go.infratographer.com/x/otelx"
	"go.uber.org/zap/zapcore"

	"go.infratographer.com/identity-api/internal/api/httpsrv"
	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/fositex"
	"go.infratographer.com/identity-api/internal/jwks"
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
	ginx.MustViperFlags(v, flags, defaultListen)
	otelx.MustViperFlags(v, flags)
}

func serve(ctx context.Context) {
	err := otelx.InitTracer(config.Config.OTel, appName, logger)
	if err != nil {
		logger.Fatalf("error initializing tracing: %s", err)
	}

	storageEngine, err := storage.NewEngine(config.Config.Storage)
	if err != nil {
		logger.Fatalf("error initializing storage: %s", err)
	}

	defer storageEngine.Shutdown()

	mappingStrategy := rfc8693.NewClaimMappingStrategy(storageEngine)

	jwksStrategy := jwks.NewIssuerJWKSURIStrategy(storageEngine)

	oauth2Config, err := fositex.NewOAuth2Config(config.Config.OAuth)
	if err != nil {
		logger.Fatalf("error loading config: %s", err)
	}

	oauth2Config.IssuerJWKSURIStrategy = jwksStrategy
	oauth2Config.ClaimMappingStrategy = mappingStrategy
	oauth2Config.UserInfoStrategy = storageEngine

	keyGetter := func(ctx context.Context) (any, error) {
		return oauth2Config.GetSigningKey(ctx), nil
	}

	hmacStrategy := compose.NewOAuth2HMACStrategy(oauth2Config)
	jwtStrategy := compose.NewOAuth2JWTStrategy(keyGetter, hmacStrategy, oauth2Config)
	store := fositestorage.NewExampleStore()

	provider := fositex.NewOAuth2Provider(
		oauth2Config,
		store,
		jwtStrategy,
		rfc8693.NewTokenExchangeHandler,
	)

	apiHandler, err := httpsrv.NewAPIHandler(storageEngine)
	if err != nil {
		logger.Fatal("error initializing API server: %s", err)
	}

	userInfoHandler, err := userinfo.NewHandler(storageEngine, oauth2Config)
	if err != nil {
		logger.Fatal("error initializing UserInfo handler: %s", err)
	}

	router := routes.NewRouter(logger, oauth2Config, provider)

	emptyLogFn := func(c *gin.Context) []zapcore.Field {
		return []zapcore.Field{}
	}

	// ginx doesn't allow configuration of ContextWithFallback but we need it here.
	engine := ginx.DefaultEngine(logger.Desugar(), emptyLogFn)
	engine.ContextWithFallback = true

	router.Routes(engine.Group("/"))
	apiHandler.Routes(engine.Group("/"))
	userInfoHandler.Routes(engine.Group("/"))

	srv := &http.Server{
		Addr:    config.Config.Server.Listen,
		Handler: engine,
	}

	logger.Fatal(srv.ListenAndServe())
}
