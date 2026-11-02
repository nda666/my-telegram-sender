package models

import "time"

type Log struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  *uint     `gorm:"index" json:"deviceId"`
	Level     string    `gorm:"size:20;default:info" json:"level"`
	Action    string    `gorm:"size:50" json:"action"`
	Message   string    `gorm:"type:text" json:"message"`
	IP        string    `gorm:"size:50" json:"ip"`
	Data      string    `gorm:"type:longtext" json:"data"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}
