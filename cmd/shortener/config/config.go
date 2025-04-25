package config

import (
	"flag"
	"os"
	"strings"
)

// Config содержит настройки приложения
type Config struct {
	RunAddr string
	BaseURL string
}

// NewConfig создает и возвращает новый объект Config с настройками по умолчанию и парсит флаги командной строки
func NewConfig() *Config {
	cfg := &Config{
		RunAddr: ":8080",
		BaseURL: "http://localhost:8080",
	}

	// Регистрируем флаги
	flagRunAddr := flag.String("a", ":8080", "address and port to run server")
	flagBaseURL := flag.String("b", "http://localhost:8080", "base URL for shortened links")
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

	// Валидация значений
	if !strings.Contains(cfg.RunAddr, ":") {
		cfg.RunAddr = ":" + cfg.RunAddr
	}
	if !strings.HasPrefix(cfg.BaseURL, "http://") && !strings.HasPrefix(cfg.BaseURL, "https://") {
		cfg.BaseURL = "http://" + cfg.BaseURL
	}

	return cfg
}
