package config

import (
	"flag"
)

// Config содержит настройки приложения
type Config struct {
	RunAddr string
	BaseURL string
}

// NewConfig создает и возвращает новый объект Config с настройками по умолчанию и парсит флаги командной строки
func NewConfig() *Config {
	var cfg Config
	flag.StringVar(&cfg.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.Parse()
	return &cfg
}
