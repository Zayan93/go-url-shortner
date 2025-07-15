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

func NewSQLStorage(db *sql.DB) *SQLStorage {
	return &SQLStorage{DB: db}
}

// Ping проверяет соединение с базой данных
func (s *SQLStorage) Ping() error {
	return s.DB.Ping()
}
