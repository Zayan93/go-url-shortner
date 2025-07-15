package store

import "sync"

type FileStorage struct {
	filename string
	producer *Producer
	mu       sync.Mutex
}

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

func (s *FileStorage) Store(id, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := &Event{
		ShortURL:    id,
		OriginalURL: url,
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
