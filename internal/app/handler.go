package app

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-url-shortner/internal/logger"
	"go-url-shortner/internal/store"
	"io"
	"net/http"
	"strconv"
	"strings"

	// Add these for Postgres error handling
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func NewHandler(s store.URLStorage, baseURL string, sqlStorage store.SQLPinger) *Handler {
	return &Handler{
		Storage:    s,
		BaseURL:    baseURL,
		SQLStorage: sqlStorage,
	}
}

type URLResponse struct {
	ShortURL    string `json:"result,omitempty"` // omitempty чтобы пропускать незаполненные
	OriginalURL string `json:"url,omitempty"`    // будет в запросе
}

type Handler struct {
	Storage    store.URLStorage
	BaseURL    string
	SQLStorage store.SQLPinger // интерфейс, а не *SQLStorage
}

type BatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func generateID() string {
	b := make([]byte, 6) // 6 байт = ~8 символов base64
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func generateUserID() string {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (h *Handler) ensureUserID(res http.ResponseWriter, req *http.Request) (string, error) {
	cookie, err := req.Cookie("user_id")
	if err != nil {
		// Куки нет, создаем новую
		userID := generateUserID()
		logger.Log.Info("Creating new user ID", zap.String("userID", userID))
		cookie := &http.Cookie{
			Name:     "user_id",
			Value:    userID,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(res, cookie)
		return userID, nil
	}

	// Кука есть, возвращаем существующий ID
	logger.Log.Info("Using existing user ID", zap.String("userID", cookie.Value))
	return cookie.Value, nil
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

	// Используем новый метод для получения URL с информацией об удалении
	if storage, ok := h.Storage.(interface {
		GetWithDeletedFlag(string) (string, bool, bool)
	}); ok {
		originalURL, exists, isDeleted := storage.GetWithDeletedFlag(id)
		if !exists {
			http.Error(res, "not found", http.StatusBadRequest)
			return
		}
		if isDeleted {
			http.Error(res, "url deleted", http.StatusGone)
			return
		}
		res.Header().Del("Content-Encoding")
		http.Redirect(res, req, originalURL, http.StatusTemporaryRedirect)
		return
	}

	// Fallback для хранилищ, которые не поддерживают GetWithDeletedFlag
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

		userID, err_ := h.ensureUserID(res, req)
		if err_ != nil {
			http.Error(res, "failed to ensure user id", http.StatusInternalServerError)
			return
		}

		logger.Log.Info("Post Handler cookie: ", zap.String("userID", userID))

		// Читаем тело запроса в буфер
		err := json.NewDecoder(req.Body).Decode(&requestBody)
		if err != nil {
			http.Error(res, "invalid request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		originalURL := requestBody.OriginalURL

		// проверяем есть ли уже такой URL в базе
		if shortID, found := h.Storage.GetShortIDByOriginalURL(originalURL); found {
			shortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortID)
			response := URLResponse{ShortURL: shortURL}
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(res).Encode(response)
			return
		}

		id := generateID()
		logger.Log.Info("Trying to store url in SQL")

		err = h.Storage.Store(id, originalURL, userID)
		if err != nil {
			// проверяем Постгрю
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == pgerrcode.UniqueViolation {
				if sqlStorage, ok := h.Storage.(interface{ GetShortIDByOriginalURL(string) (string, bool) }); ok {
					if shortID, found := sqlStorage.GetShortIDByOriginalURL(originalURL); found {
						shortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortID)
						response := URLResponse{ShortURL: shortURL}
						res.Header().Set("Content-Type", "application/json")
						res.WriteHeader(http.StatusConflict)
						_ = json.NewEncoder(res).Encode(response)

						return
					}
				}
			}
			logger.Log.Info("Postgres store error")
			http.Error(res, "failed to store url", http.StatusInternalServerError)
			return
		}

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
		userID, err_ := h.ensureUserID(res, req)
		if err_ != nil {
			http.Error(res, "failed to ensure user id", http.StatusInternalServerError)
			return
		}

		data, _ := io.ReadAll(req.Body)

		defer req.Body.Close()

		originalURL := strings.TrimSpace(string(data))

		// Check for existing original URL for non-SQL storage
		if shortID, found := h.Storage.GetShortIDByOriginalURL(originalURL); found {
			shortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortID)
			res.Header().Set("Content-Type", "text/plain")
			res.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
			res.WriteHeader(http.StatusConflict)
			res.Write([]byte(shortURL))
			return
		}

		id := generateID()

		err := h.Storage.Store(id, originalURL, userID)
		if err != nil {
			// Check for Postgres unique violation
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == pgerrcode.UniqueViolation {
				if sqlStorage, ok := h.Storage.(interface{ GetShortIDByOriginalURL(string) (string, bool) }); ok {
					if shortID, found := sqlStorage.GetShortIDByOriginalURL(originalURL); found {
						shortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortID)
						res.Header().Set("Content-Type", "text/plain")
						res.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
						res.WriteHeader(http.StatusConflict)
						res.Write([]byte(shortURL))
						return
					}
				}
			}
			http.Error(res, "failed to store url", http.StatusInternalServerError)
			return
		}

		shortURL := fmt.Sprintf("%s/%s", h.BaseURL, id)
		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(shortURL))
	} else {
		res.WriteHeader(http.StatusBadRequest)
	}
}

// ServePing проверяет соединение с базой данных
func (h *Handler) ServePing(res http.ResponseWriter, req *http.Request) {
	if h.SQLStorage == nil {
		http.Error(res, "no database configured", http.StatusInternalServerError)
		return
	}
	if err := h.SQLStorage.Ping(); err != nil {
		http.Error(res, "db connection error", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("pong"))
}

func (h *Handler) PostShortenBatch(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, err_ := h.ensureUserID(res, req)
	if err_ != nil {
		http.Error(res, "failed to ensure user id", http.StatusInternalServerError)
		return
	}

	var batch []BatchRequestItem
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&batch); err != nil {
		http.Error(res, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(batch) == 0 {
		http.Error(res, "empty batch", http.StatusBadRequest)
		return
	}

	pairs := make(map[string]string, len(batch))
	idToCorrelation := make(map[string]string, len(batch))
	for _, item := range batch {
		id := generateID()
		pairs[id] = item.OriginalURL
		idToCorrelation[id] = item.CorrelationID
	}

	if err := h.Storage.StoreBatch(pairs, userID); err != nil {
		http.Error(res, "failed to store batch", http.StatusInternalServerError)
		return
	}

	response := make([]BatchResponseItem, 0, len(batch))
	for id := range pairs {
		response = append(response, BatchResponseItem{
			CorrelationID: idToCorrelation[id],
			ShortURL:      fmt.Sprintf("%s/%s", h.BaseURL, id),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(response)
}

func (h *Handler) GetUserURLs(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := h.ensureUserID(res, req)
	if err != nil {
		logger.Log.Error("Failed to ensure user ID", zap.Error(err))
		http.Error(res, "failed to ensure user id", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Getting URLs for user", zap.String("userID", userID))

	// Получаем все URL пользователя
	urls, err := h.Storage.GetURLsByUser(userID)
	if err != nil {
		logger.Log.Error("Failed to get user URLs", zap.Error(err))
		http.Error(res, "failed to get user urls", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Found URLs for user", zap.String("userID", userID), zap.Int("count", len(urls)))

	// Если у пользователя нет URL, возвращаем 204
	if len(urls) == 0 {
		logger.Log.Info("No URLs found for user, returning 204", zap.String("userID", userID))
		res.WriteHeader(http.StatusNoContent)
		http.Error(res, "No URLs found for user", http.StatusNoContent)
	}

	logger.Log.Info("URLs found, returning 200", zap.String("userID", userID), zap.Int("count", len(urls)))
	for i, url := range urls {
		logger.Log.Info("URL found", zap.Int("index", i), zap.String("shortURL", url.ShortURL), zap.String("originalURL", url.OriginalURL))
	}

	// Преобразуем данные для ответа
	response := make([]store.URLPair, len(urls))
	for i, url := range urls {
		response[i] = store.URLPair{
			ShortURL:    fmt.Sprintf("%s/%s", h.BaseURL, url.ShortURL),
			OriginalURL: url.OriginalURL,
		}
	}

	// Возвращаем список URL в формате JSON
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(response); err != nil {
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(res, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) DeleteUserURLs(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := h.ensureUserID(res, req)
	if err != nil {
		logger.Log.Error("Failed to ensure user ID", zap.Error(err))
		http.Error(res, "failed to ensure user id", http.StatusInternalServerError)
		return
	}

	var shortIDs []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&shortIDs); err != nil {
		http.Error(res, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(shortIDs) == 0 {
		http.Error(res, "empty request", http.StatusBadRequest)
		return
	}

	// Анонимную функцию для удаления URL сделали
	go func() {
		if err := h.Storage.DeleteURLs(shortIDs, userID); err != nil {
			logger.Log.Error("Failed to delete URLs", zap.Error(err), zap.Strings("shortIDs", shortIDs), zap.String("userID", userID))
		} else {
			logger.Log.Info("Successfully deleted URLs", zap.Strings("shortIDs", shortIDs), zap.String("userID", userID))
		}
	}()

	res.WriteHeader(http.StatusAccepted)
}
