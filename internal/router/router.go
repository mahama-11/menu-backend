package router

import (
	"menu-service/internal/config"
	"menu-service/internal/middleware"
	auth "menu-service/internal/modules/auth"
	authz "menu-service/internal/modules/authz"
	referral "menu-service/internal/modules/referral"
	share "menu-service/internal/modules/share"
	studio "menu-service/internal/modules/studio"
	user "menu-service/internal/modules/user"
	"menu-service/internal/platform"
	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func New(cfg config.Config, platformClient *platform.Client, authHandler *auth.Handler, userHandler *user.Handler, authzHandler *authz.Handler, referralHandler *referral.Handler, studioHandler *studio.Handler, shareHandler *share.Handler, authzService *authz.Service) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	serviceName := cfg.Monitoring.Tracing.ServiceName
	if serviceName == "" {
		serviceName = "menu-service"
	}
	r.Use(otelgin.Middleware(serviceName))
	r.Use(middleware.RequestContext(), middleware.Metrics(cfg.Monitoring.Metrics.Namespace, cfg.Monitoring.Metrics.Subsystem), middleware.AccessLog(), gin.Recovery())

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
		v1.GET("/referrals/codes/:code/resolve", referralHandler.ResolveCode)
	}

	protected := v1.Group("")
	protected.Use(middleware.PlatformJWTAuth(cfg.Platform.JWTSecret))
	{
		protected.GET("/user/credits", userHandler.GetCredits)
		protected.GET("/user/wallet-summary", userHandler.GetWalletSummary)
		protected.GET("/user/wallet-history", userHandler.GetWalletHistory)
		protected.GET("/user/audit-history", middleware.RequireMenuPermission(authzService, "menu.audit.read"), userHandler.GetAuditHistory)
		protected.GET("/user/profile", userHandler.GetProfile)
		protected.PATCH("/user/profile", userHandler.UpdateProfile)
		protected.GET("/user/activities", userHandler.ListActivities)
		protected.GET("/studio/assets", middleware.RequireMenuPermission(authzService, "menu.asset.read"), studioHandler.ListAssets)
		protected.GET("/studio/library/assets", middleware.RequireMenuPermission(authzService, "menu.asset.read"), studioHandler.AssetLibrary)
		protected.POST("/studio/assets", middleware.RequireMenuPermission(authzService, "menu.asset.upload"), studioHandler.RegisterAsset)
		protected.GET("/studio/styles", middleware.RequireMenuPermission(authzService, "menu.template.read"), studioHandler.ListStylePresets)
		protected.POST("/studio/styles", middleware.RequireMenuPermission(authzService, "menu.template.manage"), studioHandler.CreateStylePreset)
		protected.GET("/studio/styles/:styleID", middleware.RequireMenuPermission(authzService, "menu.template.read"), studioHandler.GetStylePreset)
		protected.POST("/studio/styles/:styleID/fork", middleware.RequireMenuPermission(authzService, "menu.template.manage"), studioHandler.ForkStylePreset)
		protected.GET("/studio/jobs", middleware.RequireMenuPermission(authzService, "menu.job.read"), studioHandler.ListGenerationJobs)
		protected.GET("/studio/history/jobs", middleware.RequireMenuPermission(authzService, "menu.job.read"), studioHandler.JobHistory)
		protected.POST("/studio/jobs", middleware.RequireMenuPermission(authzService, "menu.job.create"), studioHandler.CreateGenerationJob)
		protected.GET("/studio/jobs/:jobID", middleware.RequireMenuPermission(authzService, "menu.job.read"), studioHandler.GetGenerationJob)
		protected.POST("/studio/jobs/:jobID/results", middleware.RequireMenuPermission(authzService, "menu.job.manage"), studioHandler.RecordJobResults)
		protected.POST("/studio/jobs/:jobID/cancel", middleware.RequireMenuPermission(authzService, "menu.job.manage"), studioHandler.CancelGenerationJob)
		protected.POST("/studio/jobs/:jobID/select-variant", middleware.RequireMenuPermission(authzService, "menu.job.create"), studioHandler.SelectVariant)
		protected.GET("/referrals/programs", middleware.RequireMenuPermission(authzService, "menu.referral.read"), referralHandler.ListPrograms)
		protected.GET("/referrals/me/overview", middleware.RequireMenuPermission(authzService, "menu.referral.read"), referralHandler.Overview)
		protected.GET("/referrals/me/codes", middleware.RequireMenuPermission(authzService, "menu.referral.read"), referralHandler.ListCodes)
		protected.POST("/referrals/me/codes/ensure", middleware.RequireMenuPermission(authzService, "menu.referral.manage"), referralHandler.EnsureCode)
		protected.POST("/referrals/me/codes", middleware.RequireMenuPermission(authzService, "menu.referral.manage"), referralHandler.CreateCode)
		protected.GET("/referrals/me/conversions", middleware.RequireMenuPermission(authzService, "menu.referral.read"), referralHandler.ListConversions)
		protected.GET("/referrals/me/commissions", middleware.RequireMenuPermission(authzService, "menu.referral.read"), referralHandler.ListCommissions)
		protected.POST("/referrals/me/commissions/redeem", middleware.RequireMenuPermission(authzService, "menu.referral.manage"), referralHandler.RedeemCommissions)
		protected.GET("/share/posts", middleware.RequireMenuPermission(authzService, "menu.share.read"), shareHandler.ListPosts)
		protected.POST("/share/posts", middleware.RequireMenuPermission(authzService, "menu.share.manage"), shareHandler.CreatePost)
		protected.GET("/share/posts/:shareID", middleware.RequireMenuPermission(authzService, "menu.share.read"), shareHandler.GetPost)

		protected.GET("/access/me", authzHandler.Me)
		protected.GET("/capabilities/editor", middleware.RequireMenuPermission(authzService, "menu.job.create"), func(c *gin.Context) {
			response.JSONSuccess(c, gin.H{"capability": "menu.job.create"})
		})
	}

	internal := r.Group("/internal/v1/menu")
	internal.Use(middleware.RequireInternalService(cfg.Security.ServiceSecretKey))
	{
		internal.POST("/studio/jobs/:jobID/runtime", studioHandler.InternalUpdateJobRuntime)
		internal.POST("/studio/jobs/:jobID/results", studioHandler.InternalRecordJobResults)
	}
	return r
}
