package store

import (
	"fmt"
	"sync"
)

type FileStorage struct {
	filename string
	producer *Producer
	mu       sync.Mutex
}

var _ URLStorage = (*FileStorage)(nil)

func NewFileStorage(filename string) (*FileStorage, error) {
	prod, err := NewProducer(filename)
	if err != nil {
		return nil, err
	}
	return &FileStorage{
		filename: filename,
		producer: prod,
	}, nil
}

func (s *FileStorage) Store(id, url string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := &Event{
		ShortURL:    id,
		OriginalURL: url,
		UserID:      userID,
	}
	return s.producer.WriteEvent(event)
}

func (s *FileStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	consumer, err := NewConsumer(s.filename)
	if err != nil {
		return "", false
	}
	defer consumer.Close()

	var originalURL string
	var isDeleted bool
	var found bool

	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			return "", false
		}
		if event == nil {
			break
		}
		if event.ShortURL == id {
			originalURL = event.OriginalURL
			isDeleted = event.DeletedFlag
			found = true
			// Продолжаем читать, чтобы найти последнее состояние URL
		}
	}

	if !found {
		return "", false
	}

	// Если URL помечен как удаленный, возвращаем false
	if isDeleted {
		return "", false
	}

	return originalURL, true
}

func (s *FileStorage) Close() error {
	return s.producer.Close()
}

// StoreBatch сохраняет множество сокращённых URL атомарно
func (s *FileStorage) StoreBatch(pairs map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, url := range pairs {
		event := &Event{
			ShortURL:    id,
			OriginalURL: url,
			UserID:      userID,
		}
		if err := s.producer.WriteEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStorage) GetShortIDByOriginalURL(url string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	consumer, err := NewConsumer(s.filename)
	if err != nil {
		return "", false
	}
	defer consumer.Close()

	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			return "", false
		}
		if event == nil {
			break
		}
		if event.OriginalURL == url {
			return event.ShortURL, true
		}
	}
	return "", false
}

func (s *FileStorage) GetURLsByUser(userID string) ([]URLPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	consumer, err := NewConsumer(s.filename)
	if err != nil {
		return nil, err
	}
	defer consumer.Close()

	// Используем map для отслеживания последнего состояния каждого URL
	urlStates := make(map[string]*Event)

	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			return nil, err
		}
		if event == nil {
			break
		}
		if event.UserID == userID {
			// Сохраняем последнее состояние URL
			urlStates[event.ShortURL] = event
		}
	}

	// Фильтруем только неудаленные URL
	urls := make([]URLPair, 0)
	for _, event := range urlStates {
		if !event.DeletedFlag {
			urls = append(urls, URLPair{
				ShortURL:    fmt.Sprintf("http://localhost:8080/%s", event.ShortURL),
				OriginalURL: event.OriginalURL,
			})
		}
	}

	return urls, nil
}

// DeleteURLs помечает URL как удаленные для указанного пользователя
func (s *FileStorage) DeleteURLs(shortIDs []string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, shortID := range shortIDs {
		event := &Event{
			ShortURL:    shortID,
			UserID:      userID,
			DeletedFlag: true,
		}
		if err := s.producer.WriteEvent(event); err != nil {
			return err
		}
	}

	return nil
}

// GetWithDeletedFlag возвращает URL с информацией о том, удален ли он
func (s *FileStorage) GetWithDeletedFlag(id string) (string, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	consumer, err := NewConsumer(s.filename)
	if err != nil {
		return "", false, false
	}
	defer consumer.Close()

	var originalURL string
	var isDeleted bool
	var found bool

	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			return "", false, false
		}
		if event == nil {
			break
		}
		if event.ShortURL == id {
			originalURL = event.OriginalURL
			isDeleted = event.DeletedFlag
			found = true
			// Продолжаем читать, чтобы найти последнее состояние URL
		}
	}

	return originalURL, found, isDeleted
}
