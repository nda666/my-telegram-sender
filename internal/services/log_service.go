package services

import (
	"context"
	"time"

	"github.com/tiar/telegram-sender/internal/logger"
	"github.com/tiar/telegram-sender/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LogService struct {
	db *gorm.DB
}

// 2. Buat variabel global yang diekspos (Diawali huruf kapital)
var Log *LogService

func NewLogService(db *gorm.DB) *LogService {
	Log = &LogService{db: db}
	return Log
}

func (s *LogService) WriteDetail(level, action, message, ip, data string, deviceID *uint) {
	entry := models.Log{
		Level:    level,
		Action:   action,
		Message:  message,
		IP:       ip,
		Data:     data,
		DeviceID: deviceID,
	}
	_ = s.db.Create(&entry).Error
}

func (s *LogService) Write(level, action, message string, deviceID *uint) {
	s.WriteDetail(level, action, message, "", "", deviceID)
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

func (s *LogService) ListByDevice(deviceID uint, page, perPage int) ([]models.Log, int64, error) {
	var logs []models.Log
	var total int64

	q := s.db.Model(&models.Log{}).Where("device_id = ?", deviceID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := q.Order("id DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// CleanOldLogs menghapus log yang usianya lebih lama dari retentionDays (default: 31 hari)
func (s *LogService) CleanOldLogs(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	res := s.db.Where("created_at < ?", cutoff).Delete(&models.Log{})
	if res.Error != nil {
		if logger.Log != nil {
			logger.Log.Error("gagal membersihkan log lama", zap.Error(res.Error))
		}
		return 0, res.Error
	}

	if res.RowsAffected > 0 && logger.Log != nil {
		logger.Log.Info("otomatis membersihkan log lama",
			zap.Int64("deleted_rows", res.RowsAffected),
			zap.Int("retention_days", retentionDays),
			zap.Time("cutoff", cutoff),
		)
	}

	return res.RowsAffected, nil
}

// StartAutoCleanup menjalankan pembersihan berkala di background goroutine
func (s *LogService) StartAutoCleanup(ctx context.Context, retentionDays int, interval time.Duration) {
	go func() {
		// Jalankan sekali saat startup
		_, _ = s.CleanOldLogs(retentionDays)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.CleanOldLogs(retentionDays)
			}
		}
	}()
}

