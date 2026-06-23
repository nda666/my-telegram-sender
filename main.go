package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/tiar/telegram-sender/docs"

	"github.com/joho/godotenv"

	"github.com/tiar/telegram-sender/internal/auth"
	"github.com/tiar/telegram-sender/internal/config"
	"github.com/tiar/telegram-sender/internal/database"
	"github.com/tiar/telegram-sender/internal/handlers"
	"github.com/tiar/telegram-sender/internal/inertia"
	"github.com/tiar/telegram-sender/internal/logger"
	"github.com/tiar/telegram-sender/internal/models"
	"github.com/tiar/telegram-sender/internal/routes"
	"github.com/tiar/telegram-sender/internal/services"
	"github.com/tiar/telegram-sender/internal/telegram"
	"go.uber.org/zap"
)

//go:embed all:public
var PublicFS embed.FS

//go:embed resources/views/root.html
var ViewFS embed.FS

//go:embed migrations/*.sql migrations/atlas.sum
var migrationFiles embed.FS

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger.Init(cfg.AppEnv)
	defer logger.Log.Sync()
	if cfg.AppEnv == "" {
		logger.Log.Fatal("APP_ENV is required")
	}
	if cfg.AppID == 0 || cfg.AppHash == "" {
		logger.Log.Fatal("APP_ID and APP_HASH must be set (get from https://my.telegram.org)")
	}

	if len(os.Args) > 1 && os.Args[1] == "--migrate-hash" {
		database.MigrateHashGen()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--migrate" {
		logger.Log.Info(cfg.DSN)
		if err := database.RunMigration(cfg.DSN, migrationFiles); err != nil {
			logger.Log.Fatal("migartion failed", zap.Error(err))
		}

		logger.Log.Info("migration completed")
		return
	}

	db, err := database.Connect(cfg.DSN)
	if err != nil {
		logger.Log.Fatal("failed to connect database", zap.Error(err))
	}

	i := inertia.New(cfg, PublicFS, ViewFS)
	sessions := auth.NewSessionManager(cfg.CookieKey)
	logSvc := services.NewLogService(db)
	userSvc := services.NewUserService(db)
	deviceSvc := services.NewDeviceService(db, logSvc)
	tgSvc := telegram.NewService(cfg, db, deviceSvc, logSvc)

	if cfg.AppEnv == "development" {
		db.AutoMigrate(&models.Device{}, &models.Log{}, &models.User{})
	}

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
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", 8000)),
	))
	routes.Register(mux, h, i, sessions, userSvc)

	log.Printf("listening on http://%s%s", "localhost", cfg.Addr)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0%s", cfg.Addr), mux); err != nil {
		log.Fatal(err)
	}
}
