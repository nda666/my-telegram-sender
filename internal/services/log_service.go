package services

import (
	"github.com/tiar/telegram-sender/internal/models"
	"gorm.io/gorm"
)

type LogService struct {
	db *gorm.DB
}

func NewLogService(db *gorm.DB) *LogService {
	return &LogService{db: db}
}

func (s *LogService) Write(level, action, message string, deviceID *uint) {
	entry := models.Log{
		Level:    level,
		Action:   action,
		Message:  message,
		DeviceID: deviceID,
	}
	_ = s.db.Create(&entry).Error
}

func (s *LogService) List(page, perPage int) ([]models.Log, int64, error) {
	var logs []models.Log
	var total int64

	q := s.db.Model(&models.Log{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := q.Order("id DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
