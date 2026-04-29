package app

import (
	"context"
	"fmt"
	"strings"

	"menu-service/internal/config"
	audit "menu-service/internal/modules/audit"
	auth "menu-service/internal/modules/auth"
	authz "menu-service/internal/modules/authz"
	channel "menu-service/internal/modules/channel"
	referral "menu-service/internal/modules/referral"
	share "menu-service/internal/modules/share"
	studio "menu-service/internal/modules/studio"
	templatecenter "menu-service/internal/modules/templatecenter"
	user "menu-service/internal/modules/user"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
	"menu-service/internal/router"
	"menu-service/internal/storage"
	"menu-service/internal/telemetry"
	"menu-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type App struct {
	Config   config.Config
	Router   *gin.Engine
	Platform *platform.Client
	DB       *gorm.DB
	Redis    *redis.Client
	Shutdown func(context.Context) error
}

func New(configFile string) (*App, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := validateDatabaseConfig(cfg.Database); err != nil {
		return nil, err
	}
	db, err := storage.InitDB(cfg.Database, cfg.GinMode)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	redisClient, err := storage.InitRedis(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}
	logger.Init(cfg.LogLevel, cfg.Monitoring.Tracing.ServiceName)
	shutdown, err := telemetry.InitTracing(cfg.Monitoring.Tracing)
	if err != nil {
		return nil, fmt.Errorf("init tracing: %w", err)
	}
	platformClient := platform.New(cfg.Platform)
	if err := validateStudioConfig(cfg.App, cfg.Studio); err != nil {
		return nil, err
	}
	authzRepo := repository.NewAuthzRepository(db)
	userRepo := repository.NewUserRepository(db)
	commercialRepo := repository.NewCommercialRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	shareRepo := repository.NewShareRepository(db)
	templateRepo := repository.NewTemplateCenterRepository(db)
	auditService := audit.NewService(auditRepo)
	authzService := authz.NewService(authzRepo, platformClient)
	authService := auth.NewService(platformClient, userRepo, authzService, cfg.App)
	userService := user.NewService(userRepo, commercialRepo, studioRepo, platformClient, authService, auditService)
	channelService := channel.NewService(platformClient)
	referralService := referral.NewService(platformClient, cfg.App)
	studioService := studio.NewService(studioRepo, templateRepo, shareRepo, userRepo, auditService, platformClient, cfg.App, cfg.Studio, cfg.Security)
	shareService := share.NewService(shareRepo, studioRepo, cfg.App)
	templateService := templatecenter.NewService(templateRepo, studioRepo, auditService, platformClient)
	if bootstrapErr := authzService.Bootstrap(); bootstrapErr != nil {
		return nil, fmt.Errorf("bootstrap menu authz: %w", bootstrapErr)
	}
	if referralBootstrapErr := referralService.Bootstrap(); referralBootstrapErr != nil {
		return nil, fmt.Errorf("bootstrap menu referral: %w", referralBootstrapErr)
	}
	if templateBootstrapErr := templateService.Bootstrap(); templateBootstrapErr != nil {
		return nil, fmt.Errorf("bootstrap menu template center: %w", templateBootstrapErr)
	}
	app := &App{Config: *cfg, Platform: platformClient, DB: db, Redis: redisClient, Shutdown: func(ctx context.Context) error {
		if shutdown != nil {
			return shutdown(ctx)
		}
		return nil
	}}
	app.Router = router.New(*cfg, platformClient, auth.NewHandler(authService, auditService), user.NewHandler(userService, auditService), authz.NewHandler(authzService), channel.NewHandler(channelService), referral.NewHandler(referralService, auditService), studio.NewHandler(studioService, auditService), templatecenter.NewHandler(templateService, auditService), share.NewHandler(shareService, auditService), authzService)
	return app, nil
}

func validateStudioConfig(appCfg config.AppConfig, cfg config.StudioConfig) error {
	_ = appCfg
	if cfg.ProductCode == "" {
		return fmt.Errorf("studio.product_code is required")
	}
	if cfg.ResourceType == "" {
		return fmt.Errorf("studio.resource_type is required")
	}
	if cfg.SingleBillableItem == "" || cfg.RefinementBillableItem == "" || cfg.VariationBillableItem == "" {
		return fmt.Errorf("studio billable items are required")
	}
	return nil
}

func validateDatabaseConfig(cfg config.DatabaseConfig) error {
	if strings.EqualFold(cfg.Driver, "sqlite") &&
		(cfg.Host != "database" || cfg.Port != 5432 || cfg.User != "menu" || cfg.DBName != "menu") {
		return fmt.Errorf("database.driver=sqlite but external database fields are configured; set database.driver=postgres to use host=%s dbname=%s", cfg.Host, cfg.DBName)
	}
	return nil
}
