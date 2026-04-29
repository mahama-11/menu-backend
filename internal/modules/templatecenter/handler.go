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

func (h *Handler) Meta(c *gin.Context) {
	response.JSONSuccess(c, h.service.Meta())
}

func (h *Handler) ListCatalog(c *gin.Context) {
	items, err := h.service.ListCatalogs(c.GetString("userID"), c.GetString("orgID"), ListCatalogInput{
		Cuisine:  c.Query("cuisine"),
		DishType: c.Query("dish_type"),
		Platform: c.Query("platform"),
		Mood:     c.Query("mood"),
		Query:    c.Query("query"),
		Plan:     c.Query("plan"),
	})
	if err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load template catalog", "MENU_TEMPLATE_CATALOG_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) Detail(c *gin.Context) {
	item, err := h.service.GetCatalogDetail(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID"), c.Query("plan"))
	if err != nil {
		writeTemplateError(c, err, "MENU_TEMPLATE_DETAIL_FAILED", "Failed to load template detail")
		return
	}
	response.JSONSuccess(c, item)
}

func (h *Handler) ListFavorites(c *gin.Context) {
	items, err := h.service.ListFavorites(c.GetString("userID"), c.GetString("orgID"), c.Query("plan"))
	if err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to load favorite templates", "MENU_TEMPLATE_FAVORITE_LIST_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"items": items})
}

func (h *Handler) SetFavorite(c *gin.Context) {
	if err := h.service.SetFavorite(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID")); err != nil {
		writeTemplateError(c, err, "MENU_TEMPLATE_FAVORITE_SET_FAILED", "Failed to favorite template")
		return
	}
	response.JSONSuccess(c, gin.H{"favorited": true})
}

func (h *Handler) RemoveFavorite(c *gin.Context) {
	if err := h.service.RemoveFavorite(c.GetString("userID"), c.GetString("orgID"), c.Param("templateID")); err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeInternalError, "Failed to remove favorite", "MENU_TEMPLATE_FAVORITE_REMOVE_FAILED", "Refresh and try again.")
		return
	}
	response.JSONSuccess(c, gin.H{"favorited": false})
}

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
