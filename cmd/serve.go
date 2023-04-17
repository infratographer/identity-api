package cmd

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite/compose"
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
	"go.infratographer.com/identity-api/internal/oauth2"
	"go.infratographer.com/identity-api/internal/rfc8693"
	"go.infratographer.com/identity-api/internal/routes"
	"go.infratographer.com/identity-api/internal/storage"
	"go.infratographer.com/identity-api/internal/userinfo"

	"github.com/metal-toolbox/auditevent/ginaudit"
	audithelpers "github.com/metal-toolbox/auditevent/helpers"
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

func serve(_ context.Context) {
	err := otelx.InitTracer(config.Config.OTel, appName, logger)
	if err != nil {
		logger.Fatalf("error initializing tracing: %s", err)
	}

	var engineOpts []storage.EngineOption

	if config.Config.OTel.Enabled {
		engineOpts = append(engineOpts, storage.WithTracing(config.Config.CRDB))
	}

	auditpath := viper.GetString("audit.log.path")
	if auditpath == "" {
		logger.Fatal("failed starting server. Audit log file path can't be empty")
	}

	// WARNING: This will block until the file is available;
	// make sure an initContainer creates the file
	auf, auerr := audithelpers.OpenAuditLogFileUntilSuccess(auditpath)
	if auerr != nil {
		logger.Fatalf("couldn't open audit file. error: %s", auerr)
	}
	defer auf.Close()

	auditMiddleware := ginaudit.NewJSONMiddleware(appName, auf)

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

	router := routes.NewRouter(logger, oauth2Config, provider, auditMiddleware)

	emptyLogFn := func(c *gin.Context) []zapcore.Field {
		return []zapcore.Field{}
	}

	// ginx doesn't allow configuration of ContextWithFallback but we need it here.
	engine := ginx.DefaultEngine(logger.Desugar(), emptyLogFn)
	engine.ContextWithFallback = true

	router.Routes(engine.Group("/"))

	// audit generated api endpoints through the router group
	apiGroup := engine.Group("/")
	apiGroup.Use(auditMiddleware.Audit())
	apiHandler.Routes(apiGroup)

	userInfoHandler.Routes(engine.Group("/"))

	srv := &http.Server{
		Addr:    config.Config.Server.Listen,
		Handler: engine,
	}

	logger.Fatal(srv.ListenAndServe())
}
