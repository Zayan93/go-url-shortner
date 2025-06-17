package main

import (
	"github.com/go-chi/chi/v5"
	"go-url-shortner/internal/app"
	"log"
	"net/http"
)

func main() {
	r := chi.NewRouter()
	r.Post("/", app.PostPage)
	r.Get("/{id}", app.GetPage)

	log.Fatal(http.ListenAndServe(":8080", r))
}
