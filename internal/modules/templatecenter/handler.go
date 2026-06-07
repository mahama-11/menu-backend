package templatecenter

import (
	"errors"
	"strings"

	audit "menu-service/internal/modules/audit"
	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
	audit   *audit.Service
}

func NewHandler(service *Service, auditService *audit.Service) *Handler {
	return &Handler{service: service, audit: auditService}
}

// Meta godoc
// @Summary Get Template Center metadata
// @Description Return Template Center filter metadata such as cuisines, platforms, moods, and plans.
// @Tags TemplateCenter
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Router /api/v1/menu/template-center/meta [get]
func (h *Handler) Meta(c *gin.Context) {
	response.JSONSuccess(c, h.service.Meta())
}

// ListCatalog godoc
// @Summary List Template Center catalog
// @Description List restaurant marketing templates with structured business inputs and target outputs.
// @Tags TemplateCenter
// @Produce json
// @Param cuisine query string false "Cuisine filter"
// @Param dish_type query string false "Dish type filter"
// @Param platform query string false "Target platform filter"
// @Param mood query string false "Mood filter"
// @Param query query string false "Search query"
// @Param plan query string false "Plan filter"
// @Success 200 {object} response.SuccessResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/catalog [get]
func (h *Handler) ListCatalog(c *gin.Context) {
	items, err := h.service.ListCatalogs(c.GetString("userID"), c.GetString("orgID"), ListCatalogInput{
		Cuisine:  c.Query("cuisine"),
		DishType: c.Query("dish_type"),
		Platform: c.Query("platform"),
		Mood:     c.Query("mood"),
		Query:    c.Query("query"),
		Plan:     c.Query("plan"),
		Source:   c.Query("source"),
	})
	if err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load template catalog", "MENU_TEMPLATE_CATALOG_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

// Detail godoc
// @Summary Get Template Center catalog detail
// @Description Get one template with input_slots, target_outputs, strategy_policy, and execution profile.
// @Tags TemplateCenter
// @Produce json
// @Param templateID path string true "Template ID"
// @Param plan query string false "Plan context"
// @Success 200 {object} response.SuccessResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/catalog/{templateID} [get]
func (h *Handler) Detail(c *gin.Context) {
	var (
		item *TemplateCatalogDetail
		err  error
	)
	if strings.EqualFold(strings.TrimSpace(c.Query("source")), "local") {
		item, err = h.service.GetLocalCatalogDetail(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID"), c.Query("plan"))
	} else {
		item, err = h.service.GetCatalogDetail(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID"), c.Query("plan"))
	}
	if err != nil {
		writeTemplateError(c, err, "MENU_TEMPLATE_DETAIL_FAILED", "Failed to load template detail")
		return
	}
	response.JSONSuccess(c, item)
}

// ListFavorites godoc
// @Summary List favorite templates
// @Description List current organization/user favorite Template Center items.
// @Tags TemplateCenter
// @Produce json
// @Security BearerAuth
// @Param plan query string false "Plan context"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/favorites [get]
func (h *Handler) ListFavorites(c *gin.Context) {
	items, err := h.service.ListFavorites(c.GetString("userID"), c.GetString("orgID"), c.Query("plan"))
	if err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load favorite templates", "MENU_TEMPLATE_FAVORITE_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

// SetFavorite godoc
// @Summary Favorite a template
// @Description Mark a Template Center item as favorite for the current org/user.
// @Tags TemplateCenter
// @Produce json
// @Security BearerAuth
// @Param templateID path string true "Template ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/favorites/{templateID} [post]
func (h *Handler) SetFavorite(c *gin.Context) {
	if err := h.service.SetFavorite(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID")); err != nil {
		writeTemplateError(c, err, "MENU_TEMPLATE_FAVORITE_SET_FAILED", "Failed to favorite template")
		return
	}
	response.JSONSuccess(c, gin.H{"favorited": true})
}

// RemoveFavorite godoc
// @Summary Remove template favorite
// @Description Remove a Template Center favorite for the current org/user.
// @Tags TemplateCenter
// @Produce json
// @Security BearerAuth
// @Param templateID path string true "Template ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/favorites/{templateID} [delete]
func (h *Handler) RemoveFavorite(c *gin.Context) {
	if err := h.service.RemoveFavorite(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID")); err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to remove favorite", "MENU_TEMPLATE_FAVORITE_REMOVE_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"favorited": false})
}

// Use godoc
// @Summary Prepare template usage
// @Description Resolve Template Center context into a prefilled Studio job with strategy and role-aware source assets.
// @Tags TemplateCenter
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param templateID path string true "Template ID"
// @Param plan query string false "Plan context"
// @Param request body UseTemplateInput true "Template use request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/catalog/{templateID}/use [post]
func (h *Handler) Use(c *gin.Context) {
	var req UseTemplateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONBindError(c, err, "invalid use template request")
		return
	}
	item, err := h.service.UseTemplate(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID"), c.Query("plan"), req)
	if err != nil {
		writeTemplateError(c, err, "MENU_TEMPLATE_USE_FAILED", "Failed to prepare template usage")
		return
	}
	response.JSONSuccess(c, item)
}

// CopyToMyTemplates godoc
// @Summary Copy catalog template to my templates
// @Description Copy a Template Center catalog item into an editable Studio style preset.
// @Tags TemplateCenter
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param templateID path string true "Template ID"
// @Param request body CopyTemplateInput true "Copy template request"
// @Success 201 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/template-center/catalog/{templateID}/copy-to-my-templates [post]
func (h *Handler) CopyToMyTemplates(c *gin.Context) {
	var req CopyTemplateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONBindError(c, err, "invalid copy template request")
		return
	}
	item, err := h.service.CopyToMyTemplates(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID"), req)
	if err != nil {
		writeTemplateError(c, err, "MENU_TEMPLATE_COPY_FAILED", "Failed to copy template")
		return
	}
	response.JSONSuccessWithStatus(c, 201, item)
}

func writeTemplateError(c *gin.Context, err error, errorCode, message string) {
	_ = c.Error(err)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.JSONErrorSemantic(c, response.CodeNotFound, "Template not found", errorCode, "Choose another template and try again.")
	case err != nil && (strings.Contains(err.Error(), "requires") || strings.Contains(err.Error(), "support target platform")):
		response.JSONErrorSemantic(c, response.CodeConflict, message, errorCode, err.Error())
	default:
		response.JSONErrorSemantic(c, response.CodeInternalError, message, errorCode, "Refresh and try again.")
	}
}
