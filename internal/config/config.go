package config

import (
	"flag"
	"os"
)

type Config struct {
	Address         string // адрес запуска HTTP-сервера, например localhost:8080
	BaseURL         string // базовый URL для сокращённых ссылок, например http://localhost:8080
	LogLevel        string // Уровень логирования
	FileStoragePath string // путь до файла с данными
}

// New создает и инициализирует конфигурацию из флагов командной строки
func New() *Config {
	defaultAddress := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"
	defaultLogLevel := "info"
	defaultFileStoragePath := "./storage.txt"

	envAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	envLogLevel := os.Getenv("LOG_LEVEL")
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH")

	if envAddress == "" {
		envAddress = defaultAddress
	}
	if envBaseURL == "" {
		envBaseURL = defaultBaseURL
	}

	if envLogLevel == "" {
		envLogLevel = defaultLogLevel
	}

	if envFileStoragePath == "" {
		envFileStoragePath = defaultFileStoragePath
	}

	addr := flag.String("a", envAddress, "HTTP server address")
	baseURL := flag.String("b", envBaseURL, "Base URL for short links")
	logLevel := flag.String("l", envLogLevel, "Log level")
	fileStoragePath := flag.String("f", envFileStoragePath, "File storage path")
	flag.Parse()

	return &Config{
		Address:         *addr,
		BaseURL:         *baseURL,
		LogLevel:        *logLevel,
		FileStoragePath: *fileStoragePath,
	}
}
