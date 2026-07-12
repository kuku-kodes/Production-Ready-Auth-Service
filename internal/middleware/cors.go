package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS(origins []string) gin.HandlerFunc {
	config := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			if len(origins) == 1 && origins[0] == "*" {
				return true
			}
			for _, allowed := range origins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
		MaxAge: 12 * time.Hour,
	}

	return cors.New(config)
}