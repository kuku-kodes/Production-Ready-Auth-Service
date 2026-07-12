package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kaushlender/auth-service/internal/cache"
	"github.com/kaushlender/auth-service/internal/token"
)

const (
	ContextKeyUserID    = "user_id"
	ContextKeyEmail     = "email"
	ContextKeyRole      = "role"
	ContextKeyTokenID   = "token_id"
	ContextKeyUserClaims = "user_claims"
)

type AuthMiddleware struct {
	tokenManager *token.Manager
	cache        *cache.Cache
}

func NewAuthMiddleware(tokenManager *token.Manager, cache *cache.Cache) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
		cache:        cache,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		accessToken := parts[1]

		// Validate access token
		claims, err := m.tokenManager.ValidateAccessToken(accessToken)
		if err != nil {
			if err == token.ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "token has expired",
				})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted
		isBlacklisted, err := m.cache.IsTokenBlacklisted(c.Request.Context(), claims.ID)
		if err == nil && isBlacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "token has been revoked",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyTokenID, claims.ID)
		c.Set(ContextKeyUserClaims, claims)

		c.Next()
	}
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get(ContextKeyEmail)
	if !exists {
		return "", false
	}
	return email.(string), true
}

func GetRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(ContextKeyRole)
	if !exists {
		return "", false
	}
	return role.(string), true
}

func GetTokenID(c *gin.Context) (string, bool) {
	tokenID, exists := c.Get(ContextKeyTokenID)
	if !exists {
		return "", false
	}
	return tokenID.(string), true
}