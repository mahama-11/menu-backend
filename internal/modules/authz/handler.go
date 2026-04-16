package authz

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

func (h *Handler) Me(c *gin.Context) {
	userID := c.GetString("userID")
	orgID := c.GetString("orgID")
	ctx, err := h.service.Resolve(userID, orgID)
	if err != nil {
		_ = c.Error(err)
		response.JSONErrorSemantic(c, response.CodeForbidden, "Menu access is not available for the current organization", "MENU_ACCESS_DENIED", "Switch to an organization with Menu access or contact an administrator.")
		return
	}
	response.JSONSuccess(c, ctx)
}
