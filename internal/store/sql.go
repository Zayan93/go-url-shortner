package store

import (
	"database/sql"
)

// SQLStorage реализует интерфейс для работы с SQL-базой

type SQLPinger interface {
	Ping() error
}

type SQLStorage struct {
	DB *sql.DB
}

// Ensure SQLStorage implements URLStorage
var _ URLStorage = (*SQLStorage)(nil)

// NewSQLStorage создаёт SQLStorage и инициализирует таблицу
func NewSQLStorage(db *sql.DB) (*SQLStorage, error) {
	storage := &SQLStorage{DB: db}
	if err := storage.initTable(); err != nil {
		return nil, err
	}
	return storage, nil
}

func (s *SQLStorage) initTable() error {
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS short_urls (
			id SERIAL PRIMARY KEY,
			short_id VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT NOT NULL
		);
	`)
	return err
}

// Store сохраняет сокращённый URL
func (s *SQLStorage) Store(id, url string) error {
	_, err := s.DB.Exec(`INSERT INTO short_urls (short_id, original_url) VALUES ($1, $2) ON CONFLICT (short_id) DO NOTHING`, id, url)
	return err
}

// Get возвращает оригинальный URL по short_id
func (s *SQLStorage) Get(id string) (string, bool) {
	var url string
	err := s.DB.QueryRow(`SELECT original_url FROM short_urls WHERE short_id = $1`, id).Scan(&url)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return url, true
}

// GetShortIDByOriginalURL returns the short_id for a given original_url, if it exists.
func (s *SQLStorage) GetShortIDByOriginalURL(url string) (string, bool) {
	var shortID string
	err := s.DB.QueryRow(`SELECT short_id FROM short_urls WHERE original_url = $1`, url).Scan(&shortID)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return shortID, true
}

// Ping проверяет соединение с базой данных
func (s *SQLStorage) Ping() error {
	return s.DB.Ping()
}

// StoreBatch сохраняет множество сокращённых URL в рамках одной транзакции
func (s *SQLStorage) StoreBatch(pairs map[string]string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO short_urls (short_id, original_url) VALUES ($1, $2) ON CONFLICT (short_id) DO NOTHING`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for id, url := range pairs {
		if _, err := stmt.Exec(id, url); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
