package store

import (
	"go-url-shortner/internal/logger"
	"sync"

	"go.uber.org/zap"
)

type URLPair struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type URLData struct {
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

type InMemoryStorage struct {
	store      map[string]URLData
	storeWUser map[string][]URLPair
	mu         sync.RWMutex
}

var _ URLStorage = (*InMemoryStorage)(nil)

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		store:      make(map[string]URLData),
		storeWUser: make(map[string][]URLPair),
	}
}

func (s *InMemoryStorage) Store(id, url string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[id] = URLData{OriginalURL: url, UserID: userID, IsDeleted: false}
	s.storeWUser[userID] = append(s.storeWUser[userID], URLPair{ShortURL: id, OriginalURL: url})
	return nil
}

func (s *InMemoryStorage) Get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	urlData, exists := s.store[id]
	if !exists {
		return "", false
	}
	// Если URL помечен как удаленный, возвращаем false
	if urlData.IsDeleted {
		return "", false
	}
	return urlData.OriginalURL, true
}

// StoreBatch сохраняет множество сокращённых URL атомарно
func (s *InMemoryStorage) StoreBatch(pairs map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, url := range pairs {
		s.store[id] = URLData{OriginalURL: url, UserID: userID, IsDeleted: false}
		s.storeWUser[userID] = append(s.storeWUser[userID], URLPair{ShortURL: id, OriginalURL: url})
	}
	return nil
}

func (s *InMemoryStorage) GetShortIDByOriginalURL(url string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, urlData := range s.store {
		if urlData.OriginalURL == url {
			return id, true
		}
	}
	return "", false
}

func (s *InMemoryStorage) GetURLsByUser(userID string) ([]URLPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls, exists := s.storeWUser[userID]
	if !exists {
		logger.Log.Info("No URLs found for user in memory storage", zap.String("userID", userID))
		return []URLPair{}, nil
	}

	// Фильтруем удаленные URL
	var activeURLs []URLPair
	for _, url := range urls {
		if urlData, exists := s.store[url.ShortURL]; exists && !urlData.IsDeleted {
			activeURLs = append(activeURLs, url)
		}
	}

	logger.Log.Info("Found URLs for user in memory storage", zap.String("userID", userID), zap.Int("count", len(activeURLs)))
	return activeURLs, nil
}

// DeleteURLs помечает URL как удаленные для указанного пользователя
func (s *InMemoryStorage) DeleteURLs(shortIDs []string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, shortID := range shortIDs {
		if urlData, exists := s.store[shortID]; exists && urlData.UserID == userID {
			urlData.IsDeleted = true
			s.store[shortID] = urlData
		}
	}

	return nil
}

// GetWithDeletedFlag возвращает URL с информацией о том, удален ли он
func (s *InMemoryStorage) GetWithDeletedFlag(id string) (string, bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urlData, exists := s.store[id]
	if !exists {
		return "", false, false
	}

	return urlData.OriginalURL, true, urlData.IsDeleted
}
