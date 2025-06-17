package main

import (
	"github.com/go-chi/chi/v5"
	"go-url-shortner/internal/app"
	"go-url-shortner/internal/config"
	"log"
	"net/http"
	"net/url"
)

func BaseURLMiddleware(baseURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parsedURL, err := url.Parse(baseURL)
			if err == nil {
				r.Host = parsedURL.Host
			} else {
				r.Host = baseURL
			}
			next.ServeHTTP(w, r)

		})
	}
}

func main() {
	cfg := config.New()

	r := chi.NewRouter()

	// Добавляю baseurl в middleware чтобы не трогать логику postpage хендлера и не менять тесты

	r.Use(BaseURLMiddleware(cfg.BaseURL))
	r.Post("/", app.PostPage)
	r.Get("/{id}", app.GetPage)

	log.Fatal(http.ListenAndServe(cfg.Address, r))
}
