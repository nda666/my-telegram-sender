package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppID             int
	AppHash           string
	AppURL            string
	Addr              string
	DSN               string
	CookieKey         string
	AppEnv            string
	APIRateLimitRPS   float64
	APIRateLimitBurst int
	LogRetentionDays  int
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

	rateRPS := 5.0
	if rpsStr := os.Getenv("API_RATE_LIMIT_RPS"); rpsStr != "" {
		if val, err := strconv.ParseFloat(rpsStr, 64); err == nil && val > 0 {
			rateRPS = val
		}
	}

	rateBurst := 5
	if burstStr := os.Getenv("API_RATE_LIMIT_BURST"); burstStr != "" {
		if val, err := strconv.Atoi(burstStr); err == nil && val > 0 {
			rateBurst = val
		}
	}

	retentionDays := 31
	if retStr := os.Getenv("LOG_RETENTION_DAYS"); retStr != "" {
		if val, err := strconv.Atoi(retStr); err == nil && val > 0 {
			retentionDays = val
		}
	}

	return Config{
		AppID:             appID,
		AppHash:           os.Getenv("APP_HASH"),
		AppURL:            appURL,
		Addr:              addr,
		DSN:               dsn,
		CookieKey:         cookieKey,
		AppEnv:            appEnv,
		APIRateLimitRPS:   rateRPS,
		APIRateLimitBurst: rateBurst,
		LogRetentionDays:  retentionDays,
	}
}
