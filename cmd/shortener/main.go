package main

import (
	"github.com/go-chi/chi/v5"
	"go-url-shortner/internal/app"
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

	// Storage теперь не глобальная переменная
	storage := store.NewInMemoryStorage()

	// Baseurl передаю через dependency injection в хендлеры
	handler := app.NewHandler(storage, cfg.BaseURL)

	r := chi.NewRouter()
	// r.Use(gzip.GzipMiddleware)
	r.Use(logger.WithLogging)

	r.Post("/", handler.PostPage)
	r.Post("/api/shorten", handler.PostShorten)
	r.Get("/{id}", handler.GetPage)

	logger.Log.Info("Running server", zap.String("address", cfg.Address))

	// Используем стандартный log тк пишет сразу в stderr и завершает программу
	log.Fatal(http.ListenAndServe(cfg.Address, r))

}
