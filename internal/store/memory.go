package store

import (
	"sync"
)

type URLPair struct {
	ShortURL    string
	OriginalURL string
}

type InMemoryStorage struct {
	store      map[string]string
	storeWUser map[string][]URLPair
	mu         sync.RWMutex
}

var _ URLStorage = (*InMemoryStorage)(nil)

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		store:      make(map[string]string),
		storeWUser: make(map[string][]URLPair),
	}
}

func (s *InMemoryStorage) Store(id, url string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[id] = url
	s.storeWUser[userID] = append(s.storeWUser[userID], URLPair{ShortURL: id, OriginalURL: url})
	return nil
}

func (s *InMemoryStorage) Get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, exists := s.store[id]
	return url, exists
}

// StoreBatch сохраняет множество сокращённых URL атомарно
func (s *InMemoryStorage) StoreBatch(pairs map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, url := range pairs {
		s.store[id] = url
		s.storeWUser[userID] = append(s.storeWUser[userID], URLPair{ShortURL: id, OriginalURL: url})
	}
	return nil
}

func (s *InMemoryStorage) GetShortIDByOriginalURL(url string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, storedURL := range s.store {
		if storedURL == url {
			return id, true
		}
	}
	return "", false
}

func (s *InMemoryStorage) GetURLsByUser(userID string) ([]URLPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := make([]URLPair, 0, len(s.storeWUser[userID]))
	urls = append(urls, s.storeWUser[userID]...)
	return urls, nil
}
