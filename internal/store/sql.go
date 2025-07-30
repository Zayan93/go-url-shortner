package store

import (
	"database/sql"
	"fmt"
	"go-url-shortner/internal/logger"
	"strings"

	"go.uber.org/zap"
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
			original_url TEXT NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		logger.Log.Error("Failed to create table", zap.Error(err))
	} else {
		logger.Log.Info("Table short_urls created or already exists")
	}
	return err
}

// Store сохраняет сокращённый URL
func (s *SQLStorage) Store(id, url string, userID string) error {
	logger.Log.Info("SQL storage store")
	_, err := s.DB.Exec(`INSERT INTO short_urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (short_id) DO NOTHING`, id, url, userID)
	if err != nil {
		logger.Log.Error("Failed to store URL in SQL", zap.Error(err))
	}
	return err
}

// Get возвращает оригинальный URL по short_id
func (s *SQLStorage) Get(id string) (string, bool) {
	var url string
	var isDeleted bool
	err := s.DB.QueryRow(`SELECT original_url, is_deleted FROM short_urls WHERE short_id = $1`, id).Scan(&url, &isDeleted)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	// Если URL помечен как удаленный, возвращаем false
	if isDeleted {
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
func (s *SQLStorage) StoreBatch(pairs map[string]string, userID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO short_urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (short_id) DO NOTHING`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for id, url := range pairs {
		if _, err := stmt.Exec(id, url, userID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStorage) GetURLsByUser(userID string) ([]URLPair, error) {
	rows, err := s.DB.Query(`SELECT short_id, original_url FROM short_urls WHERE user_id = $1 AND is_deleted = FALSE ORDER BY created_at DESC`, userID)
	if err != nil {
		logger.Log.Error("Failed to query user URLs", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var pairs []URLPair
	for rows.Next() {
		var shortID, originalURL string
		if err := rows.Scan(&shortID, &originalURL); err != nil {
			logger.Log.Error("Failed to scan row", zap.Error(err))
			return nil, err
		}
		pairs = append(pairs, URLPair{
			ShortURL:    shortID,
			OriginalURL: originalURL,
		})
	}

	if err = rows.Err(); err != nil {
		logger.Log.Error("Error iterating rows", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Found URLs for user in SQL storage", zap.String("userID", userID), zap.Int("count", len(pairs)))
	return pairs, nil
}

// DeleteURLs помечает URL как удаленные для указанного пользователя
func (s *SQLStorage) DeleteURLs(shortIDs []string, userID string) error {
	if len(shortIDs) == 0 {
		return nil
	}

	// Создаем плейсхолдеры для IN запроса
	placeholders := make([]string, len(shortIDs))
	args := make([]interface{}, len(shortIDs)+1)

	for i, id := range shortIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(shortIDs)] = userID

	query := fmt.Sprintf(`
		UPDATE short_urls 
		SET is_deleted = TRUE 
		WHERE short_id IN (%s) AND user_id = $%d
	`, strings.Join(placeholders, ","), len(shortIDs)+1)

	_, err := s.DB.Exec(query, args...)
	if err != nil {
		logger.Log.Error("Failed to delete URLs", zap.Error(err))
		return err
	}

	logger.Log.Info("Successfully marked URLs as deleted", zap.Strings("shortIDs", shortIDs), zap.String("userID", userID))
	return nil
}

// GetWithDeletedFlag возвращает URL с информацией о том, удален ли он
func (s *SQLStorage) GetWithDeletedFlag(id string) (string, bool, bool) {
	var url string
	var isDeleted bool

	err := s.DB.QueryRow(`
		SELECT original_url, is_deleted 
		FROM short_urls 
		WHERE short_id = $1
	`, id).Scan(&url, &isDeleted)

	if err == sql.ErrNoRows {
		return "", false, false
	}
	if err != nil {
		logger.Log.Error("Failed to get URL with deleted flag", zap.Error(err))
		return "", false, false
	}

	return url, true, isDeleted
}
