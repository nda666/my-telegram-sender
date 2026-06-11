package services

import (
	"github.com/tiar/telegram-sender/internal/models"
	"gorm.io/gorm"
)

type DeviceService struct {
	db   *gorm.DB
	logs *LogService
}

func NewDeviceService(db *gorm.DB, logs *LogService) *DeviceService {
	return &DeviceService{db: db, logs: logs}
}

func (s *DeviceService) List() ([]models.Device, error) {
	var devices []models.Device
	if err := s.db.Order("id DESC").Find(&devices).Error; err != nil {
		return nil, err
	}
	for i := range devices {
		devices[i].Status = devices[i].DisplayStatus()
	}
	return devices, nil
}

func (s *DeviceService) Find(id uint) (*models.Device, error) {
	var device models.Device
	if err := s.db.First(&device, id).Error; err != nil {
		return nil, err
	}
	device.Status = device.DisplayStatus()
	return &device, nil
}

func (s *DeviceService) Create(name, phone string) (*models.Device, error) {
	device := models.Device{
		Name:   name,
		Phone:  phone,
		Status: models.DeviceStatusNoSession,
	}
	if err := s.db.Create(&device).Error; err != nil {
		return nil, err
	}
	s.logs.Write("info", "device.create", "Device dibuat: "+name, &device.ID)
	return &device, nil
}

func (s *DeviceService) Update(id uint, name, phone string) (*models.Device, error) {
	device, err := s.Find(id)
	if err != nil {
		return nil, err
	}
	device.Name = name
	device.Phone = phone
	if err := s.db.Save(device).Error; err != nil {
		return nil, err
	}
	s.logs.Write("info", "device.update", "Device diperbarui: "+name, &device.ID)
	return device, nil
}

func (s *DeviceService) Delete(id uint) error {
	device, err := s.Find(id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(&models.Device{}, id).Error; err != nil {
		return err
	}
	s.logs.Write("info", "device.delete", "Device dihapus: "+device.Name, nil)
	return nil
}

func (s *DeviceService) UpdateSession(id uint, sessionData []byte, tgUserID int64, firstName, lastName, avatarColor, tgPhone string) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"session_data":        sessionData,
		"avatar_color":        avatarColor,
		"telegram_user_id":    tgUserID,
		"telegram_first_name": firstName,
		"telegram_last_name":  lastName,
		"telegram_phone":      tgPhone,
		"status":              models.DeviceStatusOffline,
	}).Error
}

func (s *DeviceService) UpdateStatus(id uint, status string) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Update("status", status).Error
}

func (s *DeviceService) ClearSession(id uint) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"session_data":        nil,
		"telegram_user_id":    0,
		"telegram_first_name": "",
		"telegram_phone":      "",
		"status":              models.DeviceStatusNoSession,
	}).Error
}
