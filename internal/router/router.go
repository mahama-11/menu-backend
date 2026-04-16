package router

import (
	"menu-service/internal/config"
	"menu-service/internal/middleware"
	auth "menu-service/internal/modules/auth"
	authz "menu-service/internal/modules/authz"
	user "menu-service/internal/modules/user"
	"menu-service/internal/platform"
	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func New(cfg config.Config, platformClient *platform.Client, authHandler *auth.Handler, userHandler *user.Handler, authzHandler *authz.Handler, authzService *authz.Service) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	serviceName := cfg.Monitoring.Tracing.ServiceName
	if serviceName == "" {
		serviceName = "menu-service"
	}
	r.Use(otelgin.Middleware(serviceName))
	r.Use(middleware.RequestContext(), middleware.Metrics(), middleware.AccessLog(), gin.Recovery())

	healthHandler := func(c *gin.Context) {
		response.JSONSuccess(c, gin.H{"service": "v-menu-backend", "status": "ok", "platform_base_url": platformClient.BaseURL()})
	}
	r.GET("/healthz", healthHandler)
	r.HEAD("/healthz", healthHandler)
	if cfg.Monitoring.Metrics.Enabled {
		r.GET(cfg.Monitoring.Metrics.Path, middleware.MetricsHandler(cfg.Monitoring.Metrics.Namespace, cfg.Monitoring.Metrics.Subsystem))
	}

	v1 := r.Group("/api/v1/menu")
	{
		authAPI := v1.Group("/auth")
		{
			authAPI.POST("/register", authHandler.Register)
			authAPI.POST("/login", authHandler.Login)
			authAPI.GET("/session", middleware.PlatformJWTAuth(cfg.Platform.JWTSecret), authHandler.Session)
		}

		v1.GET("/health", func(c *gin.Context) {
			response.JSONSuccess(c, gin.H{"service": "menu-api", "status": "ok"})
		})
	}

	protected := v1.Group("")
	protected.Use(middleware.PlatformJWTAuth(cfg.Platform.JWTSecret))
	{
		protected.GET("/user/credits", userHandler.GetCredits)
		protected.GET("/user/profile", userHandler.GetProfile)
		protected.PATCH("/user/profile", userHandler.UpdateProfile)
		protected.GET("/user/activities", userHandler.ListActivities)

		protected.GET("/access/me", authzHandler.Me)
		protected.GET("/capabilities/editor", middleware.RequireMenuPermission(authzService, "menu.job.create"), func(c *gin.Context) {
			response.JSONSuccess(c, gin.H{"capability": "menu.job.create"})
		})
	}
	return r
}
