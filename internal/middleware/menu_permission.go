package middleware

import (
	authz "menu-service/internal/modules/authz"
	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequireMenuPermission(service *authz.Service, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, err := service.EnsurePermission(c.GetString("userID"), c.GetString("orgID"), permission)
		if err != nil {
			_ = c.Error(err)
			response.JSONErrorSemantic(c, response.CodeForbidden, "Permission denied", "MENU_PERMISSION_DENIED", "Your current role does not have access to this operation.")
			c.Abort()
			return
		}
		c.Set("menuAccess", ctx)
		c.Next()
	}
}
