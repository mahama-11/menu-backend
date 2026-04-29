package user

import (
	"strconv"

	audit "menu-service/internal/modules/audit"
	"menu-service/internal/platform"
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

func (h *Handler) GetQuotaSummary(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.quota_summary.get")
	defer span.End()
	result, err := h.service.QuotaSummary(c.GetString("orgID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load quota summary", "QUOTA_SUMMARY_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) GetWalletHistory(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.wallet_history.get")
	defer span.End()
	result, err := h.service.WalletHistory(c.GetString("orgID"), queryInt(c, "limit", 100))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load wallet history", "WALLET_HISTORY_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) GetAuditHistory(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.audit_history.get")
	defer span.End()
	result, err := h.service.AuditHistory(
		c.GetString("userID"),
		c.GetString("orgID"),
		c.Query("target_type"),
		c.Query("status"),
		queryInt(c, "limit", 100),
		queryInt(c, "offset", 0),
	)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load audit history", "AUDIT_HISTORY_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) GetCommercialOfferings(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_offerings.get")
	defer span.End()
	result, err := h.service.CommercialOfferings(c.GetString("orgID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load commercial offerings", "COMMERCIAL_OFFERINGS_LOAD_FAILED", "Refresh and try again. If the issue continues, check platform connectivity.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) CreateCommercialOrder(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_order.create")
	defer span.End()
	var req CreateCommercialOrderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid create commercial order request")
		return
	}
	result, err := h.service.CreateCommercialOrder(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to create commercial order", "COMMERCIAL_ORDER_CREATE_FAILED", "Check sku_code or package_code and retry.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) ListCommercialOrders(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_order.list")
	defer span.End()
	result, err := h.service.ListCommercialOrders(c.GetString("orgID"), queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to list commercial orders", "COMMERCIAL_ORDER_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) GetCommercialOrder(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_order.get")
	defer span.End()
	result, err := h.service.GetCommercialOrder(c.GetString("orgID"), c.Param("orderID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load commercial order", "COMMERCIAL_ORDER_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) ConfirmCommercialOrderPayment(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_order_payment.confirm")
	defer span.End()
	var req ConfirmCommercialOrderPaymentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid confirm commercial payment request")
		return
	}
	result, err := h.service.ConfirmCommercialOrderPayment(c.GetString("userID"), c.GetString("orgID"), c.Param("orderID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		writeConfirmCommercialOrderPaymentError(c, err)
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) AssignCommercialPackage(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_package.assign")
	defer span.End()
	var req AssignCommercialPackageInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid assign package request")
		return
	}
	result, err := h.service.AssignCommercialPackage(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to assign package", "COMMERCIAL_PACKAGE_ASSIGN_FAILED", "Check package code, target organization, and platform wallet configuration before retrying.")
		return
	}
	metrics.IncBusinessCounter("menu_commercial_package_assigned_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.commercial.package.assign",
			TargetType:    "commercial_package",
			TargetID:      result.PackageCode,
			Status:        "success",
			AfterSnapshot: result,
		})
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) SimulateCommercialConsumption(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/user-handler", "menu.user.commercial_consumption.simulate")
	defer span.End()
	var req SimulateCommercialConsumptionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid simulate consumption request")
		return
	}
	result, err := h.service.SimulateCommercialConsumption(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to simulate consumption", "COMMERCIAL_CONSUMPTION_SIMULATE_FAILED", "Check wallet balance, billable item configuration, and platform metering state before retrying.")
		return
	}
	metrics.IncBusinessCounter("menu_commercial_consumption_simulated_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.commercial.consumption.simulate",
			TargetType:    "billable_item",
			TargetID:      result.BillableItemCode,
			Status:        "success",
			AfterSnapshot: result,
		})
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

func writeConfirmCommercialOrderPaymentError(c *gin.Context, err error) {
	switch platform.ResponseCode(err) {
	case 2003:
		response.JSONErrorSemantic(c, response.CodeConflict, "Wallet balance is not enough for this purchase", "COMMERCIAL_ORDER_PAYMENT_INSUFFICIENT_BALANCE", firstNonEmpty(platform.ErrorHint(err), "Recharge your wallet balance before purchasing this package."))
		return
	}
	switch platform.HTTPStatus(err) {
	case 404:
		response.JSONErrorSemantic(c, response.CodeNotFound, "Commercial order not found", "COMMERCIAL_ORDER_NOT_FOUND", firstNonEmpty(platform.ErrorHint(err), "Refresh the page and try again."))
		return
	default:
		if platform.HTTPStatus(err) >= 500 {
			response.JSONErrorSemantic(c, response.CodeServiceUnavailable, "Commercial payment service is temporarily unavailable", "COMMERCIAL_ORDER_PAYMENT_UPSTREAM_FAILED", firstNonEmpty(platform.ErrorHint(err), "Retry in a moment. If the issue continues, contact support."))
			return
		}
	}
	response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to confirm order payment", "COMMERCIAL_ORDER_PAYMENT_CONFIRM_FAILED", "Check order state and payment parameters before retrying.")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
