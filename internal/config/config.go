package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppID     int
	AppHash   string
	AppURL    string
	Addr      string
	DSN       string
	CookieKey string
	AppEnv    string
}

func Load() Config {
	appID, _ := strconv.Atoi(os.Getenv("APP_ID"))
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8000"
	}
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:8000"
	}
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/telegram_sender?charset=utf8mb4&parseTime=True&loc=Local"
	}
	cookieKey := os.Getenv("COOKIE_KEY")
	if cookieKey == "" {
		cookieKey = "change-me-in-production"
	}

	appEnv := os.Getenv("APP_ENV")

	return Config{
		AppID:     appID,
		AppHash:   os.Getenv("APP_HASH"),
		AppURL:    appURL,
		Addr:      addr,
		DSN:       dsn,
		CookieKey: cookieKey,
		AppEnv:    appEnv,
	}
}
