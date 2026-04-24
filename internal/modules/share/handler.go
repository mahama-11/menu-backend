package share

import (
	"strconv"

	audit "menu-service/internal/modules/audit"
	"menu-service/internal/telemetry"
	"menu-service/pkg/metrics"
	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	audit   *audit.Service
}

func NewHandler(service *Service, auditService *audit.Service) *Handler {
	return &Handler{service: service, audit: auditService}
}

func (h *Handler) ListPosts(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.posts.list")
	defer span.End()
	items, err := h.service.ListPosts(c.GetString("userID"), c.GetString("orgID"), c.Query("status"), queryInt(c, "limit", 50))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load share posts", "SHARE_POST_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) CreatePost(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.create")
	defer span.End()
	var req CreatePostInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid create share post request")
		return
	}
	item, err := h.service.CreatePost(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to create share post", "SHARE_POST_CREATE_FAILED", "Check the selected asset and try again.")
		return
	}
	metrics.IncBusinessCounter("menu_share_post_created_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.share.post.create",
			TargetType:    "share_post",
			TargetID:      item.ShareID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func (h *Handler) GetPost(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.get")
	defer span.End()
	item, err := h.service.GetPost(c.GetString("orgID"), c.Param("shareID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Share post not found", "SHARE_POST_NOT_FOUND", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) ListPublicPosts(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.posts.public_list")
	defer span.End()
	items, err := h.service.ListPublicPosts(queryInt(c, "limit", 24), c.DefaultQuery("sort", "recent"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load public share feed", "SHARE_FEED_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) ListFavorites(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.posts.favorites_list")
	defer span.End()
	items, err := h.service.ListFavoritePosts(c.GetString("userID"), c.GetString("orgID"), queryInt(c, "limit", 24))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load favorites", "SHARE_FAVORITES_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) GetPublicPost(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.public_get")
	defer span.End()
	item, err := h.service.GetPublicPost(c.Param("token"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Share post not found", "SHARE_POST_NOT_FOUND", "Open another shared link.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) RecordPublicView(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.public_view")
	defer span.End()
	item, err := h.service.RecordPublicView(c.Param("token"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Share post not found", "SHARE_POST_NOT_FOUND", "Open another shared link.")
		return
	}
	metrics.IncBusinessCounter("menu_share_post_view_total")
	response.JSONSuccess(c, item)
}

func (h *Handler) GetEngagement(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.engagement_get")
	defer span.End()
	item, err := h.service.GetEngagement(c.GetString("userID"), c.GetString("orgID"), c.Param("shareID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Share post not found", "SHARE_POST_NOT_FOUND", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) SetLike(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.like_set")
	defer span.End()
	var req SetEngagementInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid share like request")
		return
	}
	item, err := h.service.SetLike(c.GetString("userID"), c.GetString("orgID"), c.Param("shareID"), req.Active)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Share post not found", "SHARE_POST_NOT_FOUND", "Refresh and try again.")
		return
	}
	metrics.IncBusinessCounter("menu_share_post_like_write_total")
	response.JSONSuccess(c, item)
}

func (h *Handler) SetFavorite(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/share-handler", "menu.share.post.favorite_set")
	defer span.End()
	var req SetEngagementInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid share favorite request")
		return
	}
	item, err := h.service.SetFavorite(c.GetString("userID"), c.GetString("orgID"), c.Param("shareID"), req.Active)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Share post not found", "SHARE_POST_NOT_FOUND", "Refresh and try again.")
		return
	}
	metrics.IncBusinessCounter("menu_share_post_favorite_write_total")
	response.JSONSuccess(c, item)
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
