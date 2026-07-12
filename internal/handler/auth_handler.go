package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaushlender/auth-service/internal/middleware"
	"github.com/kaushlender/auth-service/internal/repository"
	"github.com/kaushlender/auth-service/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "Registration details"
// @Success 201 {object} service.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ipAddress := c.ClientIP()
	result, err := h.authService.Register(c.Request.Context(), &req, ipAddress)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "validation failed",
				"errors": validationErr.Errors,
			})
			return
		}
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to register user",
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Login handles user login
// @Summary Authenticate user
// @Description Login with email and password to receive JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login credentials"
// @Success 200 {object} service.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ipAddress := c.ClientIP()
	result, err := h.authService.Login(c.Request.Context(), &req, ipAddress)
	if err != nil {
		if errors.Is(err, service.ErrAccountLocked) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "account temporarily locked. Please try again later.",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to login",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Refresh handles token refresh
// @Summary Refresh access token
// @Description Get a new access token using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.RefreshRequest true "Refresh token"
// @Success 200 {object} service.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req service.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ipAddress := c.ClientIP()
	result, err := h.authService.Refresh(c.Request.Context(), &req, ipAddress)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "refresh token has expired. Please login again.",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid refresh token",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Logout handles user logout
// @Summary Logout user
// @Description Revoke refresh tokens and blacklist access token
// @Tags Authentication
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	tokenID, _ := middleware.GetTokenID(c)
	ipAddress := c.ClientIP()

	if err := h.authService.Logout(c.Request.Context(), userID, tokenID, ipAddress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully logged out",
	})
}

// GetMe returns the current user's profile
// @Summary Get current user profile
// @Description Get the authenticated user's profile information
// @Tags Users
// @Security BearerAuth
// @Success 200 {object} service.UserResponse
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/users/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// HealthCheck returns the health status of the service
// @Summary Health check
// @Description Check if the service is running
// @Tags System
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *AuthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "auth-service",
	})
}