package channel

import (
	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CurrentBinding(c *gin.Context) {
	items, err := h.service.WithContext(c.Request.Context()).CurrentBinding(c.GetString("orgID"))
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to load current channel binding")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) Overview(c *gin.Context) {
	item, err := h.service.WithContext(c.Request.Context()).Overview(c.GetString("orgID"))
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to load channel overview")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) ListCommissions(c *gin.Context) {
	items, err := h.service.WithContext(c.Request.Context()).ListCommissions(c.GetString("orgID"), c.Query("status"))
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to load channel commissions")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) ListSettlements(c *gin.Context) {
	items, err := h.service.WithContext(c.Request.Context()).ListSettlements(c.GetString("orgID"), c.Query("status"))
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to load channel settlements")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) ListAdjustments(c *gin.Context) {
	items, err := h.service.WithContext(c.Request.Context()).ListAdjustments(c.GetString("orgID"), c.Query("status"))
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to load channel adjustments")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) CreateAdjustment(c *gin.Context) {
	var req CreateAdjustmentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONBindError(c, err, "invalid channel adjustment request")
		return
	}
	item, err := h.service.WithContext(c.Request.Context()).CreateAdjustment(c.GetString("orgID"), c.GetString("userID"), req)
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to create channel adjustment")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) Preview(c *gin.Context) {
	var req PreviewInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONBindError(c, err, "invalid channel preview request")
		return
	}
	item, err := h.service.WithContext(c.Request.Context()).Preview(c.GetString("orgID"), req)
	if err != nil {
		response.JSONError(c, response.CodeInternalError, "failed to preview channel policy resolution")
		return
	}
	response.JSONSuccess(c, item)
}
