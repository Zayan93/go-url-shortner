package main

import (
	"database/sql"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go-url-shortner/internal/app"
	"go-url-shortner/internal/compressor"
	"go-url-shortner/internal/config"
	"go-url-shortner/internal/logger"
	"go-url-shortner/internal/store"
	"go.uber.org/zap"
	"log"
	"net/http"
)

func main() {

	cfg := config.New()
	flagLogLevel := cfg.LogLevel

	if err := logger.Initialize(flagLogLevel); err != nil {
		// Используем стандартный log на случай ошибки при ините нашего лога
		log.Fatalf("failed to initialize logger: %v", err)
	}

	defer logger.Log.Sync()

	var storage store.URLStorage
	var sqlStorage store.SQLPinger

	// Попытка PostgreSQL
	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDSN)
		if err == nil {
			if err = db.Ping(); err == nil {
				psqlStorage, err := store.NewSQLStorage(db)
				if err == nil {
					logger.Log.Info("Connected to PSQL server")
					storage = psqlStorage
					sqlStorage = psqlStorage
				}
			}
		}
	}

	// Если storage не выбран — пробуем файл
	if storage == nil && cfg.FileStoragePath != "" {
		fileStorage, err := store.NewFileStorage(cfg.FileStoragePath)
		if err == nil {
			logger.Log.Info("Connected to file as storage")
			storage = fileStorage
			sqlStorage = nil
		}
	}

	// Если всё ещё не выбран — память
	if storage == nil {
		logger.Log.Info("Connected to InMemory as storage")
		storage = store.NewInMemoryStorage()
		sqlStorage = nil
	}

	if storage == nil {
		log.Fatalf("failed to initialize any storage backend")
	}

	// Baseurl передаю через dependency injection в хендлеры
	handler := app.NewHandler(storage, cfg.BaseURL, sqlStorage)

	r := chi.NewRouter()
	r.Use(compressor.GzipMiddleware)
	r.Use(logger.WithLogging)

	r.Post("/", handler.PostPage)
	r.Post("/api/shorten", handler.PostShorten)
	r.Post("/api/shorten/batch", handler.PostShortenBatch)
	r.Get("/{id}", handler.GetPage)
	r.Get("/ping", handler.ServePing)

	logger.Log.Info("Running server", zap.String("address", cfg.Address))

	// Используем стандартный log тк пишет сразу в stderr и завершает программу
	log.Fatal(http.ListenAndServe(cfg.Address, r))

}
