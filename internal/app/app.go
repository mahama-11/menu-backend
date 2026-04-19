package app

import (
	"context"
	"fmt"

	"menu-service/internal/config"
	audit "menu-service/internal/modules/audit"
	auth "menu-service/internal/modules/auth"
	authz "menu-service/internal/modules/authz"
	referral "menu-service/internal/modules/referral"
	share "menu-service/internal/modules/share"
	studio "menu-service/internal/modules/studio"
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
	authzRepo := repository.NewAuthzRepository(db)
	userRepo := repository.NewUserRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	shareRepo := repository.NewShareRepository(db)
	auditService := audit.NewService(auditRepo)
	authzService := authz.NewService(authzRepo, platformClient)
	authService := auth.NewService(platformClient, userRepo, authzService, cfg.App)
	userService := user.NewService(userRepo, studioRepo, platformClient, authService, auditService)
	referralService := referral.NewService(platformClient, cfg.App)
	studioService := studio.NewService(studioRepo, shareRepo, userRepo, auditService, platformClient, cfg.App, cfg.Studio)
	shareService := share.NewService(shareRepo, studioRepo, cfg.App)
	if bootstrapErr := authzService.Bootstrap(); bootstrapErr != nil {
		return nil, fmt.Errorf("bootstrap menu authz: %w", bootstrapErr)
	}
	if referralBootstrapErr := referralService.Bootstrap(); referralBootstrapErr != nil {
		return nil, fmt.Errorf("bootstrap menu referral: %w", referralBootstrapErr)
	}
	var studioRuntime *studio.AsynqRuntime
	if cfg.Studio.WorkerEnabled {
		studioRuntime, err = studio.NewAsynqRuntime(cfg.Redis, cfg.Studio, studioService)
		if err != nil {
			return nil, fmt.Errorf("init studio runtime: %w", err)
		}
		studioService.UseRuntime(studioRuntime, nil)
		if err := studioRuntime.Start(); err != nil {
			return nil, fmt.Errorf("start studio runtime: %w", err)
		}
	}
	app := &App{Config: *cfg, Platform: platformClient, DB: db, Redis: redisClient, Shutdown: func(ctx context.Context) error {
		if studioRuntime != nil {
			studioRuntime.Shutdown()
		}
		if shutdown != nil {
			return shutdown(ctx)
		}
		return nil
	}}
	app.Router = router.New(*cfg, platformClient, auth.NewHandler(authService, auditService), user.NewHandler(userService, auditService), authz.NewHandler(authzService), referral.NewHandler(referralService, auditService), studio.NewHandler(studioService, auditService), share.NewHandler(shareService, auditService), authzService)
	return app, nil
}
