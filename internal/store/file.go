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

	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			return "", false
		}
		if event == nil {
			break
		}
		if event.ShortURL == id {
			return event.OriginalURL, true
		}
	}
	return "", false
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

	urls := make([]URLPair, 0)
	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			return nil, err
		}
		if event == nil {
			break
		}
		if event.UserID == userID {
			urls = append(urls, URLPair{
				ShortURL:    fmt.Sprintf("http://localhost:8080/%s", event.ShortURL),
				OriginalURL: event.OriginalURL,
			})
		}
	}

	return urls, nil
}
