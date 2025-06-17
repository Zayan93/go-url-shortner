package config

import (
	"flag"
	"os"
)

type Config struct {
	Address string // адрес запуска HTTP-сервера, например localhost:8080
	BaseURL string // базовый URL для сокращённых ссылок, например http://localhost:8080
}

// New создает и инициализирует конфигурацию из флагов командной строки
func New() *Config {
	defaultAddress := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"

	envAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")

	if envAddress == "" {
		envAddress = defaultAddress
	}
	if envBaseURL == "" {
		envBaseURL = defaultBaseURL
	}

	addr := flag.String("a", envAddress, "HTTP server address")
	baseURL := flag.String("b", envBaseURL, "Base URL for short links")
	flag.Parse()

	return &Config{
		Address: *addr,
		BaseURL: *baseURL,
	}
}
