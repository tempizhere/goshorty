// Package config отвечает за конфигурацию приложения.
// Загружает настройки из флагов командной строки и переменных окружения,
// включая адрес сервера, базовый URL, пути к файлам и параметры подключения к БД.
package config

import (
	"encoding/json"
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
	TrustedSubnet   string // Доверенная подсеть в формате CIDR для доступа к внутренним API
}

// ConfigFile представляет структуру для десериализации JSON-файла конфигурации
type ConfigFile struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     bool   `json:"enable_https"`
	TrustedSubnet   string `json:"trusted_subnet"`
}

// loadConfigFile загружает конфигурацию из JSON-файла
func loadConfigFile(path string) (*ConfigFile, error) {
	if path == "" {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Файл не существует, это не ошибка
		}
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	var configFile ConfigFile
	if err := json.NewDecoder(file).Decode(&configFile); err != nil {
		return nil, err
	}

	return &configFile, nil
}

// NewConfig создает и возвращает новый объект Config с настройками по умолчанию и парсит флаги командной строки
// Поддерживает настройку через переменные окружения, флаги командной строки и JSON-файл
func NewConfig() (*Config, error) {
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
		EnableHTTPS:     false,
		TrustedSubnet:   "",
	}

	// Регистрируем флаги
	flagRunAddr := flag.String("a", ":8080", "address and port to run server")
	flagBaseURL := flag.String("b", "http://localhost:8080", "base URL for shortened links")
	flagFilePath := flag.String("f", "internal/storage/storage.json", "path to file for storing URLs")
	flagDatabaseDSN := flag.String("d", "", "database DSN for PostgreSQL")
	flagJWTSecret := flag.String("j", "default_jwt_secret", "JWT secret key")
	flagEnableHTTPS := flag.Bool("s", false, "enable HTTPS server")
	flagTrustedSubnet := flag.String("t", "", "trusted subnet CIDR for internal API access")
	flagConfigFile := flag.String("c", "", "path to configuration file")
	flagConfigFileAlt := flag.String("config", "", "path to configuration file")
	flag.Parse()

	// Определяем путь к файлу конфигурации
	configFilePath, configEnvSet := os.LookupEnv("CONFIG")
	if !configEnvSet {
		configFilePath = *flagConfigFile
	}
	if configFilePath == "" {
		configFilePath = *flagConfigFileAlt
	}

	// Загружаем конфигурацию из файла
	configFile, err := loadConfigFile(configFilePath)
	if err != nil {
		return nil, err
	}

	// Применяем значения из файла конфигурации как значения по умолчанию
	if configFile != nil {
		if configFile.ServerAddress != "" {
			cfg.RunAddr = configFile.ServerAddress
		}
		if configFile.BaseURL != "" {
			cfg.BaseURL = configFile.BaseURL
		}
		if configFile.FileStoragePath != "" {
			cfg.FileStoragePath = configFile.FileStoragePath
		}
		if configFile.DatabaseDSN != "" {
			cfg.DatabaseDSN = configFile.DatabaseDSN
		}
		cfg.EnableHTTPS = configFile.EnableHTTPS
		if configFile.TrustedSubnet != "" {
			cfg.TrustedSubnet = configFile.TrustedSubnet
		}
	}

	// Проверяем переменные окружения
	if addr, addrSet := os.LookupEnv("SERVER_ADDRESS"); addrSet {
		cfg.RunAddr = addr
	} else if *flagRunAddr != "" {
		cfg.RunAddr = *flagRunAddr
	}

	if url, urlSet := os.LookupEnv("BASE_URL"); urlSet {
		cfg.BaseURL = url
	} else if *flagBaseURL != "" {
		cfg.BaseURL = *flagBaseURL
	}

	if path, pathSet := os.LookupEnv("FILE_STORAGE_PATH"); pathSet {
		cfg.FileStoragePath = path
	} else if *flagFilePath != "" {
		cfg.FileStoragePath = *flagFilePath
	}

	if dsn, dsnSet := os.LookupEnv("DATABASE_DSN"); dsnSet {
		cfg.DatabaseDSN = dsn
	} else if *flagDatabaseDSN != "" {
		cfg.DatabaseDSN = *flagDatabaseDSN
	}

	if secret, secretSet := os.LookupEnv("JWT_SECRET"); secretSet {
		cfg.JWTSecret = secret
	} else if *flagJWTSecret != "" {
		cfg.JWTSecret = *flagJWTSecret
	}

	if enableHTTPS, httpsSet := os.LookupEnv("ENABLE_HTTPS"); httpsSet {
		cfg.EnableHTTPS = enableHTTPS == "true"
	} else {
		cfg.EnableHTTPS = *flagEnableHTTPS
	}

	if trustedSubnet, subnetSet := os.LookupEnv("TRUSTED_SUBNET"); subnetSet {
		cfg.TrustedSubnet = trustedSubnet
	} else if *flagTrustedSubnet != "" {
		cfg.TrustedSubnet = *flagTrustedSubnet
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
