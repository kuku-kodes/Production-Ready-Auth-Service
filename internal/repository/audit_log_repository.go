package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/kaushlender/auth-service/internal/model"
)

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *AuditLogRepository) FindByUserID(userID uuid.UUID, limit, offset int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	r.db.Model(&model.AuditLog{}).Where("user_id = ?", userID).Count(&total)
	result := r.db.Where("user_id = ?", userID).Limit(limit).Offset(offset).Order("created_at DESC").Find(&logs)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return logs, total, nil
}

func (r *AuditLogRepository) FindAll(limit, offset int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	r.db.Model(&model.AuditLog{}).Count(&total)
	result := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&logs)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return logs, total, nil
}