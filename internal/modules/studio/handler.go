package studio

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

func (h *Handler) ListAssets(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.assets.list")
	defer span.End()
	items, err := h.service.ListAssets(c.GetString("userID"), c.GetString("orgID"), c.Query("asset_type"), c.Query("status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load assets", "STUDIO_ASSET_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) AssetLibrary(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.library.assets")
	defer span.End()
	result, err := h.service.AssetLibrary(
		c.GetString("userID"),
		c.GetString("orgID"),
		c.Query("asset_type"),
		c.Query("status"),
		c.Query("query"),
		queryInt(c, "limit", 50),
		queryInt(c, "offset", 0),
	)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load asset library", "STUDIO_ASSET_LIBRARY_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) RegisterAsset(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.asset.register")
	defer span.End()
	var req RegisterAssetInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid register studio asset request")
		return
	}
	item, err := h.service.RegisterAsset(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to register asset", "STUDIO_ASSET_REGISTER_FAILED", "Check asset input and try again.")
		return
	}
	metrics.IncBusinessCounter("studio_asset_registered_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.asset.register",
			TargetType:    "studio_asset",
			TargetID:      item.ID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func (h *Handler) ListStylePresets(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.styles.list")
	defer span.End()
	items, err := h.service.ListStylePresets(c.GetString("orgID"), c.Query("visibility"), c.Query("status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load style presets", "STUDIO_STYLE_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) CreateStylePreset(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.style.create")
	defer span.End()
	var req CreateStylePresetInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid create style preset request")
		return
	}
	item, err := h.service.CreateStylePreset(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to create style preset", "STUDIO_STYLE_CREATE_FAILED", "Check style configuration and try again.")
		return
	}
	metrics.IncBusinessCounter("studio_style_created_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.style.create",
			TargetType:    "style_preset",
			TargetID:      item.StyleID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func (h *Handler) GetStylePreset(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.style.get")
	defer span.End()
	item, err := h.service.GetStylePreset(c.GetString("orgID"), c.Param("styleID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Style preset not found", "STUDIO_STYLE_NOT_FOUND", "Choose another style preset.")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) ForkStylePreset(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.style.fork")
	defer span.End()
	var req ForkStylePresetInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid fork style preset request")
		return
	}
	item, err := h.service.ForkStylePreset(c.GetString("userID"), c.GetString("orgID"), c.Param("styleID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to fork style preset", "STUDIO_STYLE_FORK_FAILED", "Try again with another style preset.")
		return
	}
	metrics.IncBusinessCounter("studio_style_forked_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.style.fork",
			TargetType:    "style_preset",
			TargetID:      item.StyleID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func (h *Handler) ListGenerationJobs(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.jobs.list")
	defer span.End()
	items, err := h.service.ListGenerationJobs(c.GetString("userID"), c.GetString("orgID"), c.Query("status"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load generation jobs", "STUDIO_JOB_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) JobHistory(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.history.jobs")
	defer span.End()
	result, err := h.service.JobHistory(c.GetString("userID"), c.GetString("orgID"), c.Query("status"), queryInt(c, "limit", 50), queryInt(c, "offset", 0))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load job history", "STUDIO_JOB_HISTORY_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, result)
}

func (h *Handler) CreateGenerationJob(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.job.create")
	defer span.End()
	var req CreateGenerationJobInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid create generation job request")
		return
	}
	item, err := h.service.CreateGenerationJob(c.GetString("userID"), c.GetString("orgID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		writeCreateJobError(c, err)
		return
	}
	metrics.IncBusinessCounter("studio_job_created_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.job.create",
			TargetType:    "generation_job",
			TargetID:      item.JobID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func (h *Handler) GetGenerationJob(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.job.get")
	defer span.End()
	item, err := h.service.GetGenerationJob(c.GetString("orgID"), c.Param("jobID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeNotFound, "Generation job not found", "STUDIO_JOB_NOT_FOUND", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, item)
}

func writeCreateJobError(c *gin.Context, err error) {
	switch platform.ResponseCode(err) {
	case 2001:
		response.JSONErrorSemantic(c, response.CodeConflict, "Monthly allowance is not enough for this generation", "STUDIO_BILLING_ALLOWANCE_INSUFFICIENT", firstNonEmpty(platform.ErrorHint(err), "Use recharge credits or upgrade the plan to continue."))
		return
	case 2002:
		response.JSONErrorSemantic(c, response.CodeConflict, "Credits balance is not enough for this generation", "STUDIO_BILLING_CREDITS_INSUFFICIENT", firstNonEmpty(platform.ErrorHint(err), "Recharge credits and try again."))
		return
	case 2003:
		response.JSONErrorSemantic(c, response.CodeConflict, "Wallet balance is not enough for this generation", "STUDIO_BILLING_WALLET_INSUFFICIENT", firstNonEmpty(platform.ErrorHint(err), "Recharge and try again."))
		return
	case 2006:
		response.JSONErrorSemantic(c, response.CodeServiceUnavailable, "Studio billing configuration is not ready", "STUDIO_BILLING_CONFIG_MISSING", firstNonEmpty(platform.ErrorHint(err), "Contact support to complete commercial configuration before retrying."))
		return
	}
	switch platform.HTTPStatus(err) {
	case 404:
		response.JSONErrorSemantic(c, response.CodeServiceUnavailable, "Studio billing configuration is not ready", "STUDIO_BILLING_CONFIG_MISSING", firstNonEmpty(platform.ErrorHint(err), "Contact support to complete commercial configuration before retrying."))
		return
	default:
		if platform.HTTPStatus(err) >= 500 {
			response.JSONErrorSemantic(c, response.CodeServiceUnavailable, "Studio billing service is temporarily unavailable", "STUDIO_BILLING_UPSTREAM_FAILED", firstNonEmpty(platform.ErrorHint(err), "Retry in a moment. If the issue continues, contact support."))
			return
		}
	}
	response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to create generation job", "STUDIO_JOB_CREATE_FAILED", "Check the selected assets and style preset.")
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

func (h *Handler) RecordJobResults(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.job.record_results")
	defer span.End()
	var req RecordJobResultsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid record generation job results request")
		return
	}
	item, err := h.service.RecordJobResults(c.GetString("userID"), c.GetString("orgID"), c.Param("jobID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to record job results", "STUDIO_JOB_RESULT_RECORD_FAILED", "Check the result payload and try again.")
		return
	}
	metrics.IncBusinessCounter("studio_job_results_recorded_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.job.record_results",
			TargetType:    "generation_job",
			TargetID:      item.JobID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) SelectVariant(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.job.select_variant")
	defer span.End()
	var req SelectVariantInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid select generation variant request")
		return
	}
	item, err := h.service.SelectVariant(c.GetString("userID"), c.GetString("orgID"), c.Param("jobID"), req.VariantID)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to select generation result", "STUDIO_VARIANT_SELECT_FAILED", "Choose another result and try again.")
		return
	}
	metrics.IncBusinessCounter("studio_variant_selected_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.variant.select",
			TargetType:    "generation_variant",
			TargetID:      req.VariantID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) CancelGenerationJob(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.job.cancel")
	defer span.End()
	item, err := h.service.CancelGenerationJob(c.GetString("orgID"), c.Param("jobID"))
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to cancel generation job", "STUDIO_JOB_CANCEL_FAILED", "Refresh and try again.")
		return
	}
	metrics.IncBusinessCounter("studio_job_canceled_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "studio.job.cancel",
			TargetType:    "generation_job",
			TargetID:      item.JobID,
			AfterSnapshot: item,
		})
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) InternalUpdateJobRuntime(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.internal.runtime_update")
	defer span.End()
	var req UpdateJobRuntimeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid internal studio runtime update request")
		return
	}
	item, err := h.service.UpdateJobRuntime(c.Param("jobID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to update studio job runtime", "STUDIO_RUNTIME_UPDATE_FAILED", "Check internal runtime payload.")
		return
	}
	metrics.IncBusinessCounter("studio_job_runtime_updated_total")
	response.JSONSuccess(c, item)
}

func (h *Handler) InternalRecordJobResults(c *gin.Context) {
	span := telemetry.StartGinSpan(c, "menu-service/studio-handler", "menu.studio.internal.record_results")
	defer span.End()
	var req RecordJobResultsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		response.JSONBindError(c, err, "invalid internal studio result callback request")
		return
	}
	item, err := h.service.RecordJobResultsInternal(c.Param("jobID"), req)
	if err != nil {
		span.RecordError(err)
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to record studio results", "STUDIO_INTERNAL_RESULT_FAILED", "Check internal result payload.")
		return
	}
	metrics.IncBusinessCounter("studio_job_internal_results_total")
	response.JSONSuccess(c, item)
}
