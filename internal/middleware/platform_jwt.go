package middleware

import (
	"strings"
	"time"

	"menu-service/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func PlatformJWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			_ = c.Error(jwt.ErrTokenMalformed)
			response.JSONErrorSemantic(c, response.CodeUnauthorized, "Missing authorization header", "AUTH_HEADER_MISSING", "Send Authorization: Bearer <token>.")
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			_ = c.Error(jwt.ErrTokenMalformed)
			response.JSONErrorSemantic(c, response.CodeUnauthorized, "Invalid authorization header format", "AUTH_HEADER_INVALID", "Use Authorization: Bearer <token>.")
			c.Abort()
			return
		}
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (any, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			if err != nil {
				_ = c.Error(err)
			}
			response.JSONErrorSemantic(c, response.CodeUnauthorized, "Invalid or expired token", "TOKEN_INVALID", "Sign in again to get a new token.")
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			_ = c.Error(jwt.ErrTokenMalformed)
			response.JSONErrorSemantic(c, response.CodeUnauthorized, "Invalid token claims", "TOKEN_CLAIMS_INVALID", "Sign in again to refresh your token.")
			c.Abort()
			return
		}
		if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
			_ = c.Error(jwt.ErrTokenExpired)
			response.JSONErrorSemantic(c, response.CodeUnauthorized, "Token has expired", "TOKEN_EXPIRED", "Sign in again to continue.")
			c.Abort()
			return
		}
		userID, _ := claims["user_id"].(string)
		orgID, _ := claims["org_id"].(string)
		orgRole, _ := claims["org_role"].(string)
		if userID == "" || orgID == "" {
			_ = c.Error(jwt.ErrTokenMalformed)
			response.JSONErrorSemantic(c, response.CodeUnauthorized, "Missing user or organization context in token", "TOKEN_CONTEXT_MISSING", "Sign in again to refresh your token.")
			c.Abort()
			return
		}
		c.Set("userID", userID)
		c.Set("orgID", orgID)
		c.Set("orgRole", orgRole)
		c.Next()
	}
}
