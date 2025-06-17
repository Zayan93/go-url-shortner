package config

import (
	"flag"
)

type Config struct {
	Address string // адрес запуска HTTP-сервера, например localhost:8080
	BaseURL string // базовый URL для сокращённых ссылок, например http://localhost:8080
}

// New создает и инициализирует конфигурацию из флагов командной строки
func New() *Config {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	baseURL := flag.String("b", "http://localhost:8080", "Base URL for short links")
	flag.Parse()

	return &Config{
		Address: *addr,
		BaseURL: *baseURL,
	}
}
