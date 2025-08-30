// Package config отвечает за конфигурацию приложения.
// Загружает настройки из флагов командной строки и переменных окружения,
// включая адрес сервера, базовый URL, пути к файлам и параметры подключения к БД.
package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

// Config содержит настройки приложения для сервиса сокращения URL
type Config struct {
	RunAddr         string // Адрес и порт для запуска сервера
	BaseURL         string // Базовый URL для генерации коротких ссылок
	FileStoragePath string // Путь к файлу для хранения URL
	DatabaseDSN     string // Строка подключения к базе данных PostgreSQL
	JWTSecret       string // Секретный ключ для подписи JWT токенов
	EnableHTTPS     bool   // Флаг включения HTTPS
}

// NewConfig создает и возвращает новый объект Config с настройками по умолчанию и парсит флаги командной строки
// Поддерживает настройку через переменные окружения и флаги командной строки
func NewConfig() (*Config, error) {
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
		EnableHTTPS:     false,
	}

	// Регистрируем флаги
	flagRunAddr := flag.String("a", ":8080", "address and port to run server")
	flagBaseURL := flag.String("b", "http://localhost:8080", "base URL for shortened links")
	flagFilePath := flag.String("f", "internal/storage/storage.json", "path to file for storing URLs")
	flagDatabaseDSN := flag.String("d", "", "database DSN for PostgreSQL")
	flagJWTSecret := flag.String("j", "default_jwt_secret", "JWT secret key")
	flagEnableHTTPS := flag.Bool("s", false, "enable HTTPS server")
	flag.Parse()

	// Проверяем переменные окружения
	if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
		cfg.RunAddr = addr
	} else if *flagRunAddr != "" {
		cfg.RunAddr = *flagRunAddr
	}

	if url := os.Getenv("BASE_URL"); url != "" {
		cfg.BaseURL = url
	} else if *flagBaseURL != "" {
		cfg.BaseURL = *flagBaseURL
	}

	if path := os.Getenv("FILE_STORAGE_PATH"); path != "" {
		cfg.FileStoragePath = path
	} else if *flagFilePath != "" {
		cfg.FileStoragePath = *flagFilePath
	}

	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		cfg.DatabaseDSN = dsn
	} else if *flagDatabaseDSN != "" {
		cfg.DatabaseDSN = *flagDatabaseDSN
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	} else if *flagJWTSecret != "" {
		cfg.JWTSecret = *flagJWTSecret
	}

	if enableHTTPS := os.Getenv("ENABLE_HTTPS"); enableHTTPS != "" {
		cfg.EnableHTTPS = enableHTTPS == "true"
	} else {
		cfg.EnableHTTPS = *flagEnableHTTPS
	}

	// Валидация значений
	if !strings.Contains(cfg.RunAddr, ":") {
		cfg.RunAddr = ":" + cfg.RunAddr
	}
	if !strings.HasPrefix(cfg.BaseURL, "http://") && !strings.HasPrefix(cfg.BaseURL, "https://") {
		cfg.BaseURL = "http://" + cfg.BaseURL
	}
	if cfg.FileStoragePath != "" {
		// Создаём директорию для файла, если она не существует
		dir := filepath.Dir(cfg.FileStoragePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
