package referral

import (
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

func (h *Handler) ListPrograms(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.programs.list")
	defer span.End()
	items, err := h.service.ListPrograms(c.DefaultQuery("status", "active"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load referral programs", "REFERRAL_PROGRAMS_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) Overview(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.overview.get")
	defer span.End()
	item, err := h.service.Overview(c.GetString("orgID"), c.Query("conversion_status"), c.Query("commission_status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load referral overview", "REFERRAL_OVERVIEW_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) ResolveCode(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.code.resolve")
	defer span.End()
	item, err := h.service.ResolveCode(c.Param("code"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		writeReferralPlatformError(c, err, "Failed to resolve referral code", "REFERRAL_CODE_RESOLVE_FAILED", "Check the code and try again.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) ListCodes(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.codes.list")
	defer span.End()
	items, err := h.service.ListCodes(c.GetString("orgID"), c.Query("program_code"), c.Query("status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load referral codes", "REFERRAL_CODES_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) EnsureCode(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.code.ensure")
	defer span.End()
	var req CreateCodeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid ensure referral code request")
		return
	}
	item, err := h.service.EnsureCode(c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		writeReferralPlatformError(c, err, "Failed to ensure referral code", "REFERRAL_CODE_ENSURE_FAILED", "Check the selected program and try again.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) CreateCode(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.code.create")
	defer span.End()
	var req CreateCodeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid create referral code request")
		return
	}
	item, err := h.service.CreateCode(c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		writeReferralPlatformError(c, err, "Failed to create referral code", "REFERRAL_CODE_CREATE_FAILED", "Check the selected program and try again.")
		return
	}
	metrics.IncBusinessCounter("menu_referral_code_created_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.referral.code.create",
			TargetType:    "referral_code",
			TargetID:      item.ID,
			Status:        "success",
			Details:       "menu referral code created",
			AfterSnapshot: item,
		})
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func (h *Handler) ListConversions(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.conversions.list")
	defer span.End()
	items, err := h.service.ListConversions(c.GetString("orgID"), c.Query("status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load referral conversions", "REFERRAL_CONVERSIONS_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func writeReferralPlatformError(c *gin.Context, err error, fallbackMessage, fallbackCode, fallbackHint string) {
	status := response.GetHTTPStatusCode(response.CodeInternalError)
	if platform.IsConflict(err) {
		status = 409
	} else if platform.IsNotFound(err) {
		status = 404
	} else if platform.IsUnauthorized(err) {
		status = 401
	}
	errorCode := platform.ErrorCode(err)
	if errorCode == "" {
		errorCode = fallbackCode
	}
	errorHint := platform.ErrorHint(err)
	if errorHint == "" {
		errorHint = fallbackHint
	}
	message := fallbackMessage
	switch errorCode {
	case "REFERRAL_CODE_NOT_FOUND":
		message = "Referral code not found"
	case "REFERRAL_CODE_INACTIVE":
		message = "Referral code is inactive"
	case "REFERRAL_PRODUCT_MISMATCH":
		message = "Referral code does not apply to this product"
	case "REFERRAL_ALREADY_CLAIMED":
		message = "Referral reward already claimed"
	case "REFERRAL_SELF_INVITE_BLOCKED":
		message = "Self referral is not allowed"
	case "REFERRAL_TRIGGER_NOT_ELIGIBLE":
		message = "Referral trigger is not eligible"
	case "NO_REDEEMABLE_COMMISSION":
		message = "No redeemable commission available"
	}
	response.JSONErrorWithStatusSemantic(c, response.CodeInternalError, message, errorCode, errorHint, status)
}

func (h *Handler) ListCommissions(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.commissions.list")
	defer span.End()
	items, err := h.service.ListCommissions(c.GetString("orgID"), c.Query("status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load referral commissions", "REFERRAL_COMMISSIONS_LOAD_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) RedeemCommissions(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.commissions.redeem")
	defer span.End()
	var req RedeemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid redeem referral commissions request")
		return
	}
	item, err := h.service.RedeemCommissions(c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		writeReferralPlatformError(c, err, "Failed to redeem referral commissions", "REFERRAL_COMMISSION_REDEEM_FAILED", "Earned commissions are required before redeeming credits.")
		return
	}
	metrics.IncBusinessCounter("menu_referral_commission_redeemed_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.referral.commission.redeem",
			TargetType:    "referral_commission_batch",
			TargetID:      item.RewardLedgerID,
			Status:        "success",
			Details:       "menu referral commissions redeemed to credits",
			AfterSnapshot: item,
		})
	}
	response.JSONSuccess(c, item)
}
