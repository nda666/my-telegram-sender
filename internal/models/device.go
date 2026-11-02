package models

import "time"

const (
	DeviceStatusNoSession = "no_session"
	DeviceStatusOffline   = "offline"
	DeviceStatusOnline    = "online"
)

type Device struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Name              string    `gorm:"size:100;not null" json:"name"`
	Phone             string    `gorm:"size:20" json:"phone"`
	SessionData       []byte    `gorm:"type:longblob" json:"-"`
	TelegramUserID    int64     `json:"telegramUserId"`
	TelegramFirstName string    `gorm:"size:100" json:"telegramFirstName"`
	TelegramLastName  string    `gorm:"size:100" json:"telegramLastName"`
	TelegramPhone     string    `gorm:"size:20" json:"telegramPhone"`
	AvatarColor       string    `gorm:"size:20" json:"avatarColor"`
	Status            string    `gorm:"size:20;default:no_session" json:"status"`
	ApiKey            string    `gorm:"size:255;uniqueIndex" json:"apiKey"`
	WebhookURL        string    `gorm:"size:500" json:"webhookUrl"`
	WebhookSecret     string    `gorm:"size:255" json:"webhookSecret"`
	WebhookEnabled    bool      `gorm:"default:true" json:"webhookEnabled"`
	BotToken          string    `gorm:"size:255" json:"botToken"`
	BotUsername       string    `gorm:"size:100" json:"botUsername"`
	BotFirstName      string    `gorm:"size:100" json:"botFirstName"`
	BotID             int64     `json:"botId"`
	BotEnabled        bool      `gorm:"default:false" json:"botEnabled"`
	BotSessionData          []byte    `gorm:"type:longblob" json:"-"`
	BotCommandsJSON         string    `gorm:"type:text" json:"botCommands"`
	BotMenuSource           string    `gorm:"size:20;default:manual" json:"botMenuSource"`
	BotMenuURL              string    `gorm:"size:500" json:"botMenuUrl"`
	BotRequestPhoneOnStart  bool      `gorm:"default:false" json:"botRequestPhoneOnStart"`
	BotStartMessage         string    `gorm:"type:text" json:"botStartMessage"`
	BotPhoneButtonText      string    `gorm:"size:100" json:"botPhoneButtonText"`
	BotPhoneReceivedMessage string    `gorm:"type:text" json:"botPhoneReceivedMessage"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

func (d *Device) HasSession() bool {
	return len(d.SessionData) > 0
}

func (d *Device) HasBot() bool {
	return d.BotToken != "" && d.BotID != 0
}

func (d *Device) DisplayStatus() string {
	if !d.HasSession() {
		return DeviceStatusNoSession
	}
	if d.Status == "" {
		return DeviceStatusOffline
	}
	return d.Status
}
