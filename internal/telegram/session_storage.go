package telegram

import (
	"context"
	"sync"

	"github.com/tiar/telegram-sender/internal/models"
	"gorm.io/gorm"
)

type DeviceSessionStorage struct {
	db       *gorm.DB
	deviceID uint
	mu       sync.Mutex
}

func NewDeviceSessionStorage(db *gorm.DB, deviceID uint) *DeviceSessionStorage {
	return &DeviceSessionStorage{db: db, deviceID: deviceID}
}

func (s *DeviceSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var device models.Device
	if err := s.db.WithContext(ctx).First(&device, s.deviceID).Error; err != nil {
		return nil, err
	}
	if len(device.SessionData) == 0 {
		return nil, nil
	}
	return device.SessionData, nil
}

func (s *DeviceSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", s.deviceID).Updates(map[string]any{
		"session_data": data,
		"status":       models.DeviceStatusOffline,
	}).Error
}
