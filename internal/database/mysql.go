package database

import (
	"fmt"
	"log"

	"github.com/tiar/telegram-sender/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.Log{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err := seedAdmin(db); err != nil {
		return nil, err
	}

	return db, nil
}

func seedAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		Username: "admin",
		Password: string(hash),
		Name:     "Administrator",
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	log.Println("seeded default user: admin / admin")
	return nil
}
