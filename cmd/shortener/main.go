package main

import (
	"go-url-shortner/internal/app"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.PostPage)
	mux.HandleFunc("/{id}", app.GetPage)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
