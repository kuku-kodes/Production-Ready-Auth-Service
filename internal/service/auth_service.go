package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/kaushlender/auth-service/internal/cache"
	"github.com/kaushlender/auth-service/internal/config"
	"github.com/kaushlender/auth-service/internal/logger"
	"github.com/kaushlender/auth-service/internal/model"
	"github.com/kaushlender/auth-service/internal/repository"
	"github.com/kaushlender/auth-service/internal/token"
	"github.com/kaushlender/auth-service/internal/validator"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountLocked       = errors.New("account temporarily locked due to too many failed attempts")
	ErrTokenExpired        = errors.New("token has expired")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrEmailNotVerified    = errors.New("email not verified")
)

type AuthService struct {
	userRepo        *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	auditLogRepo    *repository.AuditLogRepository
	cache           *cache.Cache
	tokenManager    *token.Manager
	cfg             *config.Config
}

func NewAuthService(
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	auditLogRepo *repository.AuditLogRepository,
	cache *cache.Cache,
	tokenManager *token.Manager,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		auditLogRepo:     auditLogRepo,
		cache:            cache,
		tokenManager:     tokenManager,
		cfg:              cfg,
	}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest, ipAddress string) (*AuthResponse, error) {
	// Validate input
	v := validator.New()
	v.ValidateName(req.Name)
	v.ValidateEmail(req.Email)
	v.ValidatePassword(req.Password)

	if v.HasErrors() {
		return nil, &ValidationError{Errors: v.Errors}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.App.BcryptCost)
	if err != nil {
		logger.Error("failed to hash password", logger.Err(err))
		return nil, errors.New("failed to process registration")
	}

	// Create user
	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		IsVerified:   false,
	}

	if err := s.userRepo.Create(user); err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			return nil, err
		}
		logger.Error("failed to create user", logger.Err(err))
		return nil, errors.New("failed to create user")
	}

	// Generate tokens
	tokenPair, err := s.tokenManager.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		logger.Error("failed to generate tokens", logger.Err(err))
		return nil, errors.New("failed to generate tokens")
	}

	// Store refresh token
	refreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshDuration),
	}

	if err := s.refreshTokenRepo.Create(refreshToken); err != nil {
		logger.Error("failed to store refresh token", logger.Err(err))
		return nil, errors.New("failed to store refresh token")
	}

	// Log audit
	s.logAudit(ctx, &user.ID, "user.register", ipAddress)

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWT.AccessDuration.Seconds()),
		User:         s.toUserResponse(user),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest, ipAddress string) (*AuthResponse, error) {
	// Check if account is locked
	isLocked, err := s.cache.IsLoginLocked(ctx, req.Email)
	if err != nil {
		logger.Error("failed to check login lock", logger.Err(err))
	}
	if isLocked {
		return nil, ErrAccountLocked
	}

	// Find user by email
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			s.incrementLoginAttempts(ctx, req.Email)
			return nil, ErrInvalidCredentials
		}
		logger.Error("failed to find user", logger.Err(err))
		return nil, errors.New("failed to process login")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.incrementLoginAttempts(ctx, req.Email)
		return nil, ErrInvalidCredentials
	}

	// Reset login attempts on successful login
	_ = s.cache.ResetLoginAttempts(ctx, req.Email)

	// Generate tokens
	tokenPair, err := s.tokenManager.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		logger.Error("failed to generate tokens", logger.Err(err))
		return nil, errors.New("failed to generate tokens")
	}

	// Revoke old refresh tokens and store new one
	_ = s.refreshTokenRepo.RevokeAllForUser(user.ID)
	refreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshDuration),
	}
	if err := s.refreshTokenRepo.Create(refreshToken); err != nil {
		logger.Error("failed to store refresh token", logger.Err(err))
		return nil, errors.New("failed to store refresh token")
	}

	// Log audit
	s.logAudit(ctx, &user.ID, "user.login", ipAddress)

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWT.AccessDuration.Seconds()),
		User:         s.toUserResponse(user),
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *RefreshRequest, ipAddress string) (*AuthResponse, error) {
	// Validate refresh token JWT
	claims, err := s.tokenManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, token.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidRefreshToken
	}

	// Find stored refresh token
	storedToken, err := s.refreshTokenRepo.FindByToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		logger.Error("failed to find refresh token", logger.Err(err))
		return nil, errors.New("failed to process refresh")
	}

	if storedToken.Revoked {
		// Token reuse detected - revoke all tokens for this user
		_ = s.refreshTokenRepo.RevokeAllForUser(claims.UserID)
		logger.Warn("refresh token reuse detected",
			logger.String("user_id", claims.UserID.String()),
			logger.String("ip", ipAddress),
		)
		return nil, ErrInvalidRefreshToken
	}

	if storedToken.IsExpired() {
		return nil, ErrTokenExpired
	}

	// Find user
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		logger.Error("failed to find user", logger.Err(err))
		return nil, errors.New("failed to process refresh")
	}

	// Revoke old refresh token
	if err := s.refreshTokenRepo.Revoke(storedToken.ID); err != nil {
		logger.Error("failed to revoke old refresh token", logger.Err(err))
		return nil, errors.New("failed to process refresh")
	}

	// Generate new tokens
	tokenPair, err := s.tokenManager.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		logger.Error("failed to generate tokens", logger.Err(err))
		return nil, errors.New("failed to generate tokens")
	}

	// Store new refresh token
	newRefreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshDuration),
	}
	if err := s.refreshTokenRepo.Create(newRefreshToken); err != nil {
		logger.Error("failed to store new refresh token", logger.Err(err))
		return nil, errors.New("failed to store refresh token")
	}

	// Log audit
	s.logAudit(ctx, &user.ID, "token.refresh", ipAddress)

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWT.AccessDuration.Seconds()),
		User:         s.toUserResponse(user),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, accessTokenID string, ipAddress string) error {
	// Revoke all refresh tokens for user
	if err := s.refreshTokenRepo.RevokeAllForUser(userID); err != nil {
		logger.Error("failed to revoke refresh tokens", logger.Err(err))
		return errors.New("failed to process logout")
	}

	// Blacklist access token
	if accessTokenID != "" {
		if err := s.cache.BlacklistToken(ctx, accessTokenID, s.cfg.JWT.AccessDuration); err != nil {
			logger.Error("failed to blacklist access token", logger.Err(err))
		}
	}

	// Log audit
	s.logAudit(ctx, &userID, "user.logout", ipAddress)

	return nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*UserResponse, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, err
		}
		logger.Error("failed to find user", logger.Err(err))
		return nil, errors.New("failed to get user")
	}

	resp := s.toUserResponse(user)
	return &resp, nil
}

func (s *AuthService) incrementLoginAttempts(ctx context.Context, email string) {
	if _, err := s.cache.IncrementLoginAttempts(ctx, email); err != nil {
		logger.Error("failed to increment login attempts", logger.Err(err))
	}
}

func (s *AuthService) logAudit(ctx context.Context, userID *uuid.UUID, action, ipAddress string) {
	auditLog := &model.AuditLog{
		UserID:    userID,
		Action:    action,
		IPAddress: ipAddress,
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		logger.Error("failed to create audit log", logger.Err(err), logger.String("action", action))
	}
}

func (s *AuthService) toUserResponse(user *model.User) UserResponse {
	return UserResponse{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt,
	}
}

// Helper to handle database operations
type ServiceError struct {
	Message    string
	StatusCode int
}

func (e *ServiceError) Error() string {
	return e.Message
}

// ValidationError represents validation errors
type ValidationError struct {
	Errors []validator.ValidationError
}

func (e *ValidationError) Error() string {
	if len(e.Errors) > 0 {
		return e.Errors[0].Message
	}
	return "validation error"
}

// Database transaction helper
func WithTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}