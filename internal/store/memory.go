package store

import "sync"

type InMemoryStorage struct {
	store map[string]string
	mu    sync.RWMutex
}

var _ URLStorage = (*InMemoryStorage)(nil)

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		store: make(map[string]string),
	}
}

func (s *InMemoryStorage) Store(id, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[id] = url
	return nil
}

func (s *InMemoryStorage) Get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, exists := s.store[id]
	return url, exists
}

// StoreBatch сохраняет множество сокращённых URL атомарно
func (s *InMemoryStorage) StoreBatch(pairs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, url := range pairs {
		s.store[id] = url
	}
	return nil
}
