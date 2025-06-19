package main

import (
	"github.com/go-chi/chi/v5"
	"go-url-shortner/internal/app"
	"go-url-shortner/internal/config"
	"go-url-shortner/internal/store"
	"log"
	"net/http"
)

func main() {

	cfg := config.New()

	// Storage теперь не глобальная переменная
	storage := store.NewInMemoryStorage()

	// Baseurl передаю через dependency injection в хендлеры
	handler := app.NewHandler(storage, cfg.BaseURL)

	r := chi.NewRouter()

	r.Post("/", handler.PostPage)
	r.Get("/{id}", handler.GetPage)

	log.Fatal(http.ListenAndServe(cfg.Address, r))
}
