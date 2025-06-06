package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

// Config содержит настройки приложения
type Config struct {
	RunAddr         string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	JWTSecret       string
}

// NewConfig создает и возвращает новый объект Config с настройками по умолчанию и парсит флаги командной строки
func NewConfig() (*Config, error) {
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
	}

	// Регистрируем флаги
	flagRunAddr := flag.String("a", ":8080", "address and port to run server")
	flagBaseURL := flag.String("b", "http://localhost:8080", "base URL for shortened links")
	flagFilePath := flag.String("f", "internal/storage/storage.json", "path to file for storing URLs")
	flagDatabaseDSN := flag.String("d", "", "database DSN for PostgreSQL")
	flagJWTSecret := flag.String("j", "default_jwt_secret", "JWT secret key")
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
