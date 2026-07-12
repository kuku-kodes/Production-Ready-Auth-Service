package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kaushlender/auth-service/internal/cache"
	"github.com/kaushlender/auth-service/internal/config"
	"github.com/kaushlender/auth-service/internal/handler"
	"github.com/kaushlender/auth-service/internal/logger"
	"github.com/kaushlender/auth-service/internal/middleware"
	"github.com/kaushlender/auth-service/internal/repository"
	"github.com/kaushlender/auth-service/internal/service"
	"github.com/kaushlender/auth-service/internal/token"
)

// @title Authentication Service API
// @version 1.0
// @description Production-ready authentication service for SaaS applications
// @termsOfService https://github.com/kaushlender/auth-service

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer: ` prefix, e.g. "Bearer abcde12345".
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize logger
	if err := logger.Init(cfg.App.LogLevel, cfg.App.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("starting authentication service",
		logger.String("environment", cfg.App.Environment),
		logger.String("port", cfg.Server.Port),
	)

	// Initialize database
	db, err := config.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to database", logger.Err(err))
	}

	// Run migrations
	if err := config.AutoMigrate(db); err != nil {
		logger.Fatal("failed to run migrations", logger.Err(err))
	}
	logger.Info("database migrations completed")

	// Initialize Redis cache
	redisCache := cache.New(&cfg.Redis)
	defer redisCache.Close()

	// Test Redis connection
	if err := redisCache.Ping(context.Background()); err != nil {
		logger.Warn("redis connection failed, caching disabled", logger.Err(err))
	} else {
		logger.Info("redis connection established")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Initialize token manager
	tokenManager := token.NewManager(&cfg.JWT)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		auditLogRepo,
		redisCache,
		tokenManager,
		cfg,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager, redisCache)
	rateLimiter := middleware.NewRateLimiter(redisCache, &cfg.App.RateLimit)

	// Setup Gin router
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(cfg.App.CORSOrigins))
	router.Use(rateLimiter.Limit())

	// Health endpoint
	router.GET("/health", authHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			// Auth routes (protected)
			protected.POST("/auth/logout", authHandler.Logout)

			// User routes
			protected.GET("/users/me", authHandler.GetMe)
		}
	}

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		logger.Info("server listening", logger.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", logger.Err(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", logger.Err(err))
	}

	logger.Info("server exited")
}