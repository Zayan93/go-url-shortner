package app

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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

type URLResponse struct {
	ShortURL    string `json:"result,omitempty"` // omitempty чтобы пропускать незаполненные
	OriginalURL string `json:"url,omitempty"`    // будет в запросе
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
	res.Header().Del("Content-Encoding")
	http.Redirect(res, req, originalURL, http.StatusTemporaryRedirect)

}

func (h *Handler) PostShorten(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		var requestBody URLResponse

		// Читаем тело запроса в буфер
		err := json.NewDecoder(req.Body).Decode(&requestBody)
		if err != nil {
			http.Error(res, "invalid request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		originalURL := requestBody.OriginalURL

		id := generateID()

		h.Storage.Store(id, originalURL)

		shortURL := fmt.Sprintf("%s/%s", h.BaseURL, id)

		response := URLResponse{ShortURL: shortURL}

		// Сереализуем обратно ответ

		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(res).Encode(response); err != nil {
			http.Error(res, "failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		res.WriteHeader(http.StatusBadRequest)
	}
}

func (h *Handler) PostPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		data, _ := io.ReadAll(req.Body)

		defer req.Body.Close()

		originalURL := strings.TrimSpace(string(data))
		id := generateID()

		h.Storage.Store(id, originalURL)

		shortURL := fmt.Sprintf("%s/%s", h.BaseURL, id)
		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(shortURL))
	} else {
		res.WriteHeader(http.StatusBadRequest)
	}
}
