package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/tiar/telegram-sender/internal/auth"
	"github.com/tiar/telegram-sender/internal/config"
	"github.com/tiar/telegram-sender/internal/database"
	"github.com/tiar/telegram-sender/internal/handlers"
	"github.com/tiar/telegram-sender/internal/inertia"
	"github.com/tiar/telegram-sender/internal/routes"
	"github.com/tiar/telegram-sender/internal/services"
	"github.com/tiar/telegram-sender/internal/telegram"
)

//go:embed all:public
var PublicFS embed.FS

//go:embed resources/views/root.html
var ViewFS embed.FS

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	if cfg.AppID == 0 || cfg.AppHash == "" {
		log.Fatal("APP_ID and APP_HASH must be set (get from https://my.telegram.org)")
	}

	db, err := database.Connect(cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}

	i := inertia.New(cfg, PublicFS, ViewFS)
	sessions := auth.NewSessionManager(cfg.CookieKey)
	logSvc := services.NewLogService(db)
	userSvc := services.NewUserService(db)
	deviceSvc := services.NewDeviceService(db, logSvc)
	tgSvc := telegram.NewService(cfg, db, deviceSvc, logSvc)

	h := &handlers.Handlers{
		Inertia:  i,
		Session:  sessions,
		Users:    userSvc,
		Devices:  deviceSvc,
		Logs:     logSvc,
		Telegram: tgSvc,
		Pending:  auth.NewPendingStore(),
	}

	mux := http.NewServeMux()

	// Serve embedded static assets
	mux.Handle("/build/", inertia.PublicHandler(PublicFS))

	routes.Register(mux, h, i, sessions)

	log.Printf("listening on http://%s%s", "localhost", cfg.Addr)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0%s", cfg.Addr), mux); err != nil {
		log.Fatal(err)
	}
}
