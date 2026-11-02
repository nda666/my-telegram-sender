package services

import (
	"strings"

	"github.com/google/uuid"
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
		ApiKey: uuid.NewString(),
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

// EnsureAPIKey memastikan device punya api_key, generate kalau belum ada.
func (s *DeviceService) EnsureAPIKey(deviceID int64) (string, error) {
	var key string
	err := s.db.Raw(`SELECT COALESCE(api_key,'') FROM devices WHERE id=?`, deviceID).Scan(&key).Error
	if err != nil {
		return "", err
	}
	if key != "" {
		return key, nil
	}
	key = uuid.NewString()
	db := s.db.Exec(`UPDATE devices SET api_key=? WHERE id=?`, key, deviceID)
	return key, db.Error
}

// FindByAPIKey mencari device berdasarkan api_key.
func (s *DeviceService) FindByAPIKey(apiKey string) (*models.Device, error) {
	var d models.Device

	err := s.db.
		Where("api_key = ?", apiKey).
		First(&d).
		Error

	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *DeviceService) UpdateWebhook(id uint, webhookURL, webhookSecret string, webhookEnabled bool) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"webhook_url":     webhookURL,
		"webhook_secret":  webhookSecret,
		"webhook_enabled": webhookEnabled,
	}).Error
}

func (s *DeviceService) UpdateBot(id uint, botToken, botUsername, botFirstName string, botID int64, botEnabled bool, botSessionData []byte) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"bot_token":        botToken,
		"bot_username":     botUsername,
		"bot_first_name":   botFirstName,
		"bot_id":           botID,
		"bot_enabled":      botEnabled,
		"bot_session_data": botSessionData,
	}).Error
}

func (s *DeviceService) ClearBot(id uint) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"bot_token":        "",
		"bot_username":     "",
		"bot_first_name":   "",
		"bot_id":           0,
		"bot_enabled":      false,
		"bot_session_data": nil,
	}).Error
}

func (s *DeviceService) ToggleBot(id uint, enabled bool) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Update("bot_enabled", enabled).Error
}

func (s *DeviceService) UpdateBotCommands(id uint, commandsJSON string) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Update("bot_commands_json", commandsJSON).Error
}

func (s *DeviceService) UpdateBotMenuConfig(id uint, menuSource, menuURL string) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"bot_menu_source": menuSource,
		"bot_menu_url":    menuURL,
	}).Error
}

func (s *DeviceService) UpdateBotStartConfig(id uint, requestPhone bool, startMessage, phoneBtnText, receivedMsg string) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]any{
		"bot_request_phone_on_start": requestPhone,
		"bot_start_message":          startMessage,
		"bot_phone_button_text":      phoneBtnText,
		"bot_phone_received_message": receivedMsg,
	}).Error
}

func (s *DeviceService) ListActiveWebhookDevices() ([]models.Device, error) {
	var allDevices []models.Device
	if err := s.db.Find(&allDevices).Error; err != nil {
		return nil, err
	}

	var active []models.Device
	for _, d := range allDevices {
		hasUserOrBot := d.HasSession() || (d.HasBot() && d.BotEnabled)
		if hasUserOrBot && d.WebhookEnabled && strings.TrimSpace(d.WebhookURL) != "" {
			active = append(active, d)
		}
	}
	return active, nil
}



