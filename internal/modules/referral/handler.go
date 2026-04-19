package referral

import (
	"errors"
	"io"
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

// ListPrograms godoc
// @Summary List referral programs
// @Description List active or filtered referral programs available to the current Menu product.
// @Tags Referral
// @Produce json
// @Security BearerAuth
// @Param status query string false "Program status filter" default(active)
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/programs [get]
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

// Overview godoc
// @Summary Get referral overview
// @Description Load current organization referral overview including programs, codes, conversions, commissions, and aggregate commission metrics.
// @Tags Referral
// @Produce json
// @Security BearerAuth
// @Param conversion_status query string false "Conversion status filter"
// @Param commission_status query string false "Commission status filter"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/overview [get]
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

// ResolveCode godoc
// @Summary Resolve referral code
// @Description Resolve a referral code before signup and return reward policy details for the current Menu product.
// @Tags Referral
// @Produce json
// @Param code path string true "Referral code"
// @Success 200 {object} response.SuccessResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/codes/{code}/resolve [get]
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

// ListCodes godoc
// @Summary List referral codes
// @Description List referral codes owned by the current organization with optional program and status filters.
// @Tags Referral
// @Produce json
// @Security BearerAuth
// @Param program_code query string false "Program code filter"
// @Param status query string false "Code status filter"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/codes [get]
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

// EnsureCode godoc
// @Summary Ensure referral code
// @Description Idempotently ensure the current organization has an active referral code for the selected program.
// @Tags Referral
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCodeInput true "Ensure referral code request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/codes/ensure [post]
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

// CreateCode godoc
// @Summary Create referral code
// @Description Create a new referral code for the current organization and selected program.
// @Tags Referral
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCodeInput true "Create referral code request"
// @Success 201 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/codes [post]
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

// ListConversions godoc
// @Summary List referral conversions
// @Description List referral conversions for the current organization with optional status filtering.
// @Tags Referral
// @Produce json
// @Security BearerAuth
// @Param status query string false "Conversion status filter"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/conversions [get]
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

// ListCommissions godoc
// @Summary List commissions
// @Description List referral commissions for the current organization with optional status filtering.
// @Tags Referral
// @Produce json
// @Security BearerAuth
// @Param status query string false "Commission status filter"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/commissions [get]
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

// RedeemCommissions godoc
// @Summary Redeem commissions
// @Description Redeem earned referral commissions into the configured Menu reward asset for the current organization.
// @Tags Referral
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RedeemInput false "Redeem commissions request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/referrals/me/commissions/redeem [post]
func (h *Handler) RedeemCommissions(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/referral-handler", "menu.referral.commissions.redeem")
	defer span.End()
	var req RedeemInput
	if err := bindOptionalJSON(c, &req); err != nil {
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

func bindOptionalJSON(c *gin.Context, target any) error {
	if err := c.ShouldBindJSON(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}
