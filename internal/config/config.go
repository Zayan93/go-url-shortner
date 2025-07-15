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
	DatabaseDSN     string // подключение к базе данных (host)
	DBName          string // название базы данных
	DBUser          string // имя пользователя БД
	DBPassword      string // пароль пользователя БД
}

// New создает и инициализирует конфигурацию из флагов командной строки
func New() *Config {
	defaultAddress := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"
	defaultLogLevel := "info"
	defaultFileStoragePath := "./storage.txt"
	defaultDBDSN := "localhost"
	defaultDBName := "videos"
	defaultDBUser := "postgres"
	defaultDBPassword := "root"

	envAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	envLogLevel := os.Getenv("LOG_LEVEL")
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	envSQLDBDSN := os.Getenv("DATABASE_DSN")
	envDBName := os.Getenv("DB_NAME")
	envDBUser := os.Getenv("DB_USER")
	envDBPassword := os.Getenv("DB_PASSWORD")

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
	if envSQLDBDSN == "" {
		envSQLDBDSN = defaultDBDSN
	}
	if envDBName == "" {
		envDBName = defaultDBName
	}
	if envDBUser == "" {
		envDBUser = defaultDBUser
	}
	if envDBPassword == "" {
		envDBPassword = defaultDBPassword
	}

	addr := flag.String("a", envAddress, "HTTP server address")
	baseURL := flag.String("b", envBaseURL, "Base URL for short links")
	logLevel := flag.String("l", envLogLevel, "Log level")
	fileStoragePath := flag.String("f", envFileStoragePath, "File storage path")
	databaseDSN := flag.String("d", envSQLDBDSN, "Database DSN for connection (host)")
	dbName := flag.String("db-name", envDBName, "Database name")
	dbUser := flag.String("db-user", envDBUser, "Database user")
	dbPassword := flag.String("db-password", envDBPassword, "Database password")
	flag.Parse()

	return &Config{
		Address:         *addr,
		BaseURL:         *baseURL,
		LogLevel:        *logLevel,
		FileStoragePath: *fileStoragePath,
		DatabaseDSN:     *databaseDSN,
		DBName:          *dbName,
		DBUser:          *dbUser,
		DBPassword:      *dbPassword,
	}
}
