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
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (d *Device) HasSession() bool {
	return len(d.SessionData) > 0
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
