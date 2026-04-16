package auth

import (
	"net/http"

	audit "menu-service/internal/modules/audit"
	"menu-service/internal/platform"
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

// Register godoc
// @Summary Register menu user
// @Description Frontend-facing register endpoint. Menu orchestrates platform user/org creation and optional signup bonus issuance.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterInput true "Register request"
// @Success 201 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 502 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONBindError(c, err, "invalid register request")
		return
	}
	result, err := h.service.Register(req)
	if err != nil {
		_ = c.Error(err)
		writePlatformError(c, err, "register failed")
		return
	}
	metrics.IncBusinessCounter("menu_auth_register_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.auth.register",
			TargetType:    "user",
			TargetID:      result.User.ID,
			Status:        "success",
			Details:       "menu register completed",
			AfterSnapshot: result.User,
		})
	}
	response.JSONSuccessWithStatus(c, http.StatusCreated, result)
}

// Login godoc
// @Summary Login menu user
// @Description Frontend-facing login endpoint backed by platform identity.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginInput true "Login request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 502 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONBindError(c, err, "invalid login request")
		return
	}
	result, err := h.service.Login(req)
	if err != nil {
		_ = c.Error(err)
		writePlatformError(c, err, "login failed")
		return
	}
	metrics.IncBusinessCounter("menu_auth_login_total")
	if h.audit != nil {
		_ = h.audit.RecordFromGin(c, audit.RecordInput{
			Action:        "menu.auth.login",
			TargetType:    "user",
			TargetID:      result.User.ID,
			Status:        "success",
			Details:       "menu login completed",
			AfterSnapshot: result.User,
		})
	}
	response.JSONSuccess(c, result)
}

// Session godoc
// @Summary Get current session
// @Description Resolve current authenticated user, current organization, and current credits balance for frontend bootstrap.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 502 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/menu/auth/session [get]
func (h *Handler) Session(c *gin.Context) {
	result, err := h.service.Session(c.GetString("userID"), c.GetString("orgID"))
	if err != nil {
		_ = c.Error(err)
		writePlatformError(c, err, "load session failed")
		return
	}
	response.JSONSuccess(c, result)
}

func writePlatformError(c *gin.Context, err error, fallback string) {
	if err == nil {
		response.JSONErrorSemantic(c, response.CodeInternalError, fallback, "INTERNAL_ERROR", "Please try again later.")
		return
	}
	status := http.StatusInternalServerError
	if platform.IsConflict(err) {
		status = http.StatusConflict
	} else if platform.IsNotFound(err) {
		status = http.StatusNotFound
	} else if platform.IsUnauthorized(err) {
		status = http.StatusUnauthorized
	}
	errorCode := "UPSTREAM_REQUEST_FAILED"
	errorHint := "Please try again later."
	message := err.Error()
	switch platform.ErrorCode(err) {
	case "INVALID_CREDENTIALS":
		errorCode = "INVALID_CREDENTIALS"
		errorHint = defaultString(platform.ErrorHint(err), "Check your email and password and try again.")
		message = "Invalid email or password"
	case "EMAIL_ALREADY_EXISTS":
		errorCode = "EMAIL_ALREADY_EXISTS"
		errorHint = defaultString(platform.ErrorHint(err), "Use another email or sign in with the existing account.")
		message = "Email already exists"
	default:
		if platform.ErrorCode(err) != "" {
			errorCode = platform.ErrorCode(err)
		}
		if platform.ErrorHint(err) != "" {
			errorHint = platform.ErrorHint(err)
		}
	}
	response.JSONErrorWithStatusSemantic(c, statusToCode(status), message, errorCode, errorHint, status)
}

func statusToCode(status int) response.ResponseCode {
	switch status {
	case http.StatusConflict:
		return response.CodeConflict
	case http.StatusNotFound:
		return response.CodeNotFound
	case http.StatusUnauthorized:
		return response.CodeUnauthorized
	default:
		return response.CodeExternalDependency
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
