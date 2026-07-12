package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kaushlender/auth-service/internal/cache"
	"github.com/kaushlender/auth-service/internal/config"
)

type RateLimiter struct {
	cache *cache.Cache
	cfg   *config.RateLimitConfig
}

func NewRateLimiter(cache *cache.Cache, cfg *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		cache: cache,
		cfg:   cfg,
	}
}

func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		window := time.Minute

		count, err := rl.cache.IncrementRateLimit(c.Request.Context(), key, window)
		if err != nil {
			c.Next()
			return
		}

		if count > rl.cfg.RequestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}