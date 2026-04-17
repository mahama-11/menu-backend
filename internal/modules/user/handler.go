package user

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

// ListActivities godoc
// @Summary List user activities
// @Description Query activity and history records for the current authenticated user.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Page size" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/user/activities [get]
func (h *Handler) ListActivities(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.activities.list")
	defer span.End()
	limit := queryInt(c, "limit", 20)
	offset := queryInt(c, "offset", 0)
	result, err := h.service.Activities(c.GetString("userID"), c.GetString("orgID"), limit, offset)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONError(c, response.CodeInternalError, "failed to list activities")
		return
	}
	response.JSONSuccess(c, result)
}

// GetProfile godoc
// @Summary Get user profile
// @Description Query current user profile and current restaurant/org view for frontend settings.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/user/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.profile.get")
	defer span.End()
	result, err := h.service.Profile(c.GetString("userID"), c.GetString("orgID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load profile", "PROFILE_LOAD_FAILED", "Refresh and try again. If the issue continues, contact support.")
		return
	}
	response.JSONSuccess(c, result)
}

// GetCredits godoc
// @Summary Get credits and plan detail
// @Description Query current credits balance and current plan view for the authenticated user.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/user/credits [get]
func (h *Handler) GetCredits(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.credits.get")
	defer span.End()
	result, err := h.service.Credits(c.GetString("userID"), c.GetString("orgID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load credits", "CREDITS_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) GetWalletSummary(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.wallet_summary.get")
	defer span.End()
	result, err := h.service.WalletSummary(c.GetString("orgID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load wallet summary", "WALLET_SUMMARY_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update current user name, restaurant name, and language preference through Menu orchestration.
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileInput true "Profile update request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/user/profile [patch]
func (h *Handler) UpdateProfile(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.profile.update")
	defer span.End()
	var req UpdateProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid update profile request")
		return
	}
	result, err := h.service.UpdateProfile(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to update profile", "PROFILE_UPDATE_FAILED", "Check the input and try again.")
		return
	}
	metrics.IncBusinessCounter("menu_user_profile_updated_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.user.profile.update",
			TargetType:    "user_profile",
			TargetID:      c.GetString("userID"),
			Status:        "success",
			Details:       "menu profile updated",
			AfterSnapshot: result,
		})
	}
	response.JSONSuccess(c, result)
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
