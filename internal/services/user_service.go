package services

import (
	"errors"
	"strings"

	"github.com/tiar/telegram-sender/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("username atau password salah")
var ErrInvalidUsers = errors.New("username salah")

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	hash := strings.TrimSpace(user.Password)
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

func (s *UserService) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpdatePassword(id uint, hashedPassword string) error {
	db := s.db.Exec(
		`UPDATE users SET password=? WHERE id=?`, hashedPassword, id,
	)

	Log.Write("info", "password", "user update password", nil)
	return db.Error
}
