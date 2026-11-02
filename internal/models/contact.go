package models

import "time"

type DeviceContact struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	DeviceID       uint      `gorm:"uniqueIndex:idx_dev_tguser" json:"deviceId"`
	TelegramUserID int64     `gorm:"uniqueIndex:idx_dev_tguser" json:"telegramUserId"`
	Phone          string    `gorm:"size:30" json:"phone"`
	FirstName      string    `gorm:"size:100" json:"firstName"`
	LastName       string    `gorm:"size:100" json:"lastName"`
	Username       string    `gorm:"size:100" json:"username"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
