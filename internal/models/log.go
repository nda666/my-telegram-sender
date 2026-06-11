package models

import "time"

type Log struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  *uint     `gorm:"index" json:"deviceId"`
	Level     string    `gorm:"size:20;default:info" json:"level"`
	Action    string    `gorm:"size:50" json:"action"`
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}
