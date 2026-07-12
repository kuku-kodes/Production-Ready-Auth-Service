package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/kaushlender/auth-service/internal/model"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenRevoked  = errors.New("refresh token has been revoked")
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *RefreshTokenRepository) FindByToken(token string) (*model.RefreshToken, error) {
	var refreshToken model.RefreshToken
	result := r.db.Where("token = ?", token).First(&refreshToken)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, result.Error
	}
	return &refreshToken, nil
}

func (r *RefreshTokenRepository) Revoke(id uuid.UUID) error {
	result := r.db.Model(&model.RefreshToken{}).Where("id = ?", id).Update("revoked", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(userID uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Update("revoked", true).Error
}

func (r *RefreshTokenRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.RefreshToken{}).Error
}

func (r *RefreshTokenRepository) FindValidByUserID(userID uuid.UUID) (*model.RefreshToken, error) {
	var refreshToken model.RefreshToken
	result := r.db.Where("user_id = ? AND revoked = ? AND expires_at > ?", userID, false, time.Now()).
		First(&refreshToken)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, result.Error
	}
	return &refreshToken, nil
}