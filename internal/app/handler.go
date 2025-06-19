package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"go-url-shortner/internal/store"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func NewHandler(s store.URLStorage, baseURL string) *Handler {
	return &Handler{
		Storage: s,
		BaseURL: baseURL,
	}
}

type Handler struct {
	Storage store.URLStorage
	BaseURL string
}

func generateID() string {
	b := make([]byte, 6) // 6 байт = ~8 символов base64
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (h *Handler) GetPage(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(res, "bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimPrefix(req.URL.Path, "/")
	if id == "" {
		http.Error(res, "bad request", http.StatusBadRequest)
		return
	}

	originalURL, found := h.Storage.Get(id)
	if !found {
		http.Error(res, "not found", http.StatusBadRequest)
		return
	}

	http.Redirect(res, req, originalURL, http.StatusTemporaryRedirect)

}

func (h *Handler) PostPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		data, _ := io.ReadAll(req.Body)

		defer req.Body.Close()

		originalURL := strings.TrimSpace(string(data))
		id := generateID()

		h.Storage.Store(id, originalURL)

		shortURL := fmt.Sprintf("http://%s/%s", h.BaseURL, id)
		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(shortURL))
	} else {
		res.WriteHeader(http.StatusBadRequest)
	}
}
