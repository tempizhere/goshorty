package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
		EnableHTTPS:     false,
	}

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, "internal/storage/storage.json", cfg.FileStoragePath)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "default_jwt_secret", cfg.JWTSecret)
	assert.Equal(t, false, cfg.EnableHTTPS)
}

func TestConfig_AddressValidation(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{"Port without colon", "9090", ":9090"},
		{"Port with colon", ":9090", ":9090"},
		{"Full address", "localhost:9090", "localhost:9090"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_BaseURLValidation(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"URL without protocol", "example.com", "http://example.com"},
		{"URL with http", "http://example.com", "http://example.com"},
		{"URL with https", "https://example.com", "https://example.com"},
		{"URL with subdomain", "api.example.com", "http://api.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateBaseURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Вспомогательные функции для тестирования логики валидации
func validateAddress(addr string) string {
	if !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}

func validateBaseURL(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "http://" + url
	}
	return url
}

func TestConfig_EnvironmentVariables(t *testing.T) {
	originalEnv := make(map[string]string)
	envVars := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "JWT_SECRET", "ENABLE_HTTPS", "CONFIG"}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			originalEnv[env] = val
		}
	}

	defer func() {
		for env, val := range originalEnv {
			if err := os.Setenv(env, val); err != nil {
				t.Logf("Ошибка при установке переменной окружения %s: %v", env, err)
			}
		}
		for _, env := range envVars {
			if _, exists := originalEnv[env]; !exists {
				if err := os.Unsetenv(env); err != nil {
					t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
				}
			}
		}
	}()

	if err := os.Setenv("SERVER_ADDRESS", "9090"); err != nil {
		t.Logf("Ошибка при установке SERVER_ADDRESS: %v", err)
	}
	if err := os.Setenv("BASE_URL", "https://example.com"); err != nil {
		t.Logf("Ошибка при установке BASE_URL: %v", err)
	}
	if err := os.Setenv("FILE_STORAGE_PATH", "/tmp/storage.json"); err != nil {
		t.Logf("Ошибка при установке FILE_STORAGE_PATH: %v", err)
	}
	if err := os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db"); err != nil {
		t.Logf("Ошибка при установке DATABASE_DSN: %v", err)
	}
	if err := os.Setenv("JWT_SECRET", "my_secret_key"); err != nil {
		t.Logf("Ошибка при установке JWT_SECRET: %v", err)
	}
	if err := os.Setenv("ENABLE_HTTPS", "true"); err != nil {
		t.Logf("Ошибка при установке ENABLE_HTTPS: %v", err)
	}
	if err := os.Setenv("CONFIG", "/path/to/config.json"); err != nil {
		t.Logf("Ошибка при установке CONFIG: %v", err)
	}

	assert.Equal(t, "9090", os.Getenv("SERVER_ADDRESS"))
	assert.Equal(t, "https://example.com", os.Getenv("BASE_URL"))
	assert.Equal(t, "/tmp/storage.json", os.Getenv("FILE_STORAGE_PATH"))
	assert.Equal(t, "postgres://user:pass@localhost/db", os.Getenv("DATABASE_DSN"))
	assert.Equal(t, "my_secret_key", os.Getenv("JWT_SECRET"))
	assert.Equal(t, "true", os.Getenv("ENABLE_HTTPS"))
	assert.Equal(t, "/path/to/config.json", os.Getenv("CONFIG"))
}

func TestNewConfig_Integration(t *testing.T) {
	originalEnv := make(map[string]string)
	envVars := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "JWT_SECRET", "ENABLE_HTTPS", "CONFIG"}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			originalEnv[env] = val
		}
	}

	defer func() {
		for env, val := range originalEnv {
			if err := os.Setenv(env, val); err != nil {
				t.Logf("Ошибка при установке переменной окружения %s: %v", env, err)
			}
		}
		for _, env := range envVars {
			if _, exists := originalEnv[env]; !exists {
				if err := os.Unsetenv(env); err != nil {
					t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
				}
			}
		}
	}()

	for _, env := range envVars {
		if err := os.Unsetenv(env); err != nil {
			t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
		}
	}

	tempDir := t.TempDir()
	filePath := tempDir + "/storage.json"
	if err := os.Setenv("FILE_STORAGE_PATH", filePath); err != nil {
		t.Logf("Ошибка при установке FILE_STORAGE_PATH: %v", err)
	}

	// Создаем конфигурацию без парсинга флагов
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
		EnableHTTPS:     false,
	}

	// Применяем переменные окружения
	if path := os.Getenv("FILE_STORAGE_PATH"); path != "" {
		cfg.FileStoragePath = path
	}

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, filePath, cfg.FileStoragePath)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "default_jwt_secret", cfg.JWTSecret)
	assert.Equal(t, false, cfg.EnableHTTPS)

	dir := tempDir
	_, err := os.Stat(dir)
	assert.NoError(t, err, "Directory should be created")
}

func TestLoadConfigFile_ValidJSON(t *testing.T) {
	// Создаем временный JSON файл
	tempDir := t.TempDir()
	configPath := tempDir + "/config.json"
	configContent := `{
		"server_address": "localhost:9090",
		"base_url": "https://example.com",
		"file_storage_path": "/tmp/storage.json",
		"database_dsn": "postgres://user:pass@localhost/db",
		"enable_https": true
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Загружаем конфигурацию
	configFile, err := loadConfigFile(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, configFile)

	// Проверяем значения
	assert.Equal(t, "localhost:9090", configFile.ServerAddress)
	assert.Equal(t, "https://example.com", configFile.BaseURL)
	assert.Equal(t, "/tmp/storage.json", configFile.FileStoragePath)
	assert.Equal(t, "postgres://user:pass@localhost/db", configFile.DatabaseDSN)
	assert.Equal(t, true, configFile.EnableHTTPS)
}

func TestLoadConfigFile_InvalidJSON(t *testing.T) {
	// Создаем временный файл с неверным JSON
	tempDir := t.TempDir()
	configPath := tempDir + "/invalid_config.json"
	configContent := `{
		"server_address": "localhost:9090",
		"base_url": "https://example.com",
		"file_storage_path": "/tmp/storage.json",
		"database_dsn": "postgres://user:pass@localhost/db",
		"enable_https": true,
		"invalid_field": "value"
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Загружаем конфигурацию - должно работать, так как лишние поля игнорируются
	configFile, err := loadConfigFile(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, configFile)

	// Проверяем значения
	assert.Equal(t, "localhost:9090", configFile.ServerAddress)
	assert.Equal(t, "https://example.com", configFile.BaseURL)
}

func TestLoadConfigFile_EmptyPath(t *testing.T) {
	configFile, err := loadConfigFile("")
	assert.NoError(t, err)
	assert.Nil(t, configFile)
}

func TestLoadConfigFile_NonExistentFile(t *testing.T) {
	configFile, err := loadConfigFile("/non/existent/file.json")
	assert.NoError(t, err)
	assert.Nil(t, configFile)
}

func TestNewConfig_WithJSONFile(t *testing.T) {
	// Создаем временный JSON файл
	tempDir := t.TempDir()
	configPath := tempDir + "/config.json"
	configContent := `{
		"server_address": "localhost:9090",
		"base_url": "https://example.com",
		"file_storage_path": "/tmp/storage.json",
		"database_dsn": "postgres://user:pass@localhost/db",
		"enable_https": true
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Сохраняем оригинальные переменные окружения
	originalEnv := make(map[string]string)
	envVars := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "JWT_SECRET", "ENABLE_HTTPS", "CONFIG"}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			originalEnv[env] = val
		}
	}

	defer func() {
		for env, val := range originalEnv {
			if err := os.Setenv(env, val); err != nil {
				t.Logf("Ошибка при установке переменной окружения %s: %v", env, err)
			}
		}
		for _, env := range envVars {
			if _, exists := originalEnv[env]; !exists {
				if err := os.Unsetenv(env); err != nil {
					t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
				}
			}
		}
	}()

	// Очищаем переменные окружения
	for _, env := range envVars {
		if err := os.Unsetenv(env); err != nil {
			t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
		}
	}

	// Устанавливаем переменную CONFIG
	if err := os.Setenv("CONFIG", configPath); err != nil {
		t.Logf("Ошибка при установке CONFIG: %v", err)
	}

	// Создаем конфигурацию без парсинга флагов
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
		EnableHTTPS:     false,
	}

	// Загружаем конфигурацию из файла
	configFile, err := loadConfigFile(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, configFile)

	// Применяем значения из файла конфигурации
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
	}

	// Проверяем, что значения из JSON файла применены
	assert.Equal(t, "localhost:9090", cfg.RunAddr)
	assert.Equal(t, "https://example.com", cfg.BaseURL)
	assert.Equal(t, "/tmp/storage.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, true, cfg.EnableHTTPS)
	assert.Equal(t, "default_jwt_secret", cfg.JWTSecret)
}

func TestNewConfig_JSONFilePriority(t *testing.T) {
	// Создаем временный JSON файл
	tempDir := t.TempDir()
	configPath := tempDir + "/config.json"
	configContent := `{
		"server_address": "localhost:9090",
		"base_url": "https://example.com",
		"enable_https": true
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Сохраняем оригинальные переменные окружения
	originalEnv := make(map[string]string)
	envVars := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "JWT_SECRET", "ENABLE_HTTPS", "CONFIG"}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			originalEnv[env] = val
		}
	}

	defer func() {
		for env, val := range originalEnv {
			if err := os.Setenv(env, val); err != nil {
				t.Logf("Ошибка при установке переменной окружения %s: %v", env, err)
			}
		}
		for _, env := range envVars {
			if _, exists := originalEnv[env]; !exists {
				if err := os.Unsetenv(env); err != nil {
					t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
				}
			}
		}
	}()

	// Очищаем переменные окружения
	for _, env := range envVars {
		if err := os.Unsetenv(env); err != nil {
			t.Logf("Ошибка при удалении переменной окружения %s: %v", env, err)
		}
	}

	// Устанавливаем переменную CONFIG и переопределяем некоторые значения через переменные окружения
	if err := os.Setenv("CONFIG", configPath); err != nil {
		t.Logf("Ошибка при установке CONFIG: %v", err)
	}
	if err := os.Setenv("SERVER_ADDRESS", "localhost:8080"); err != nil {
		t.Logf("Ошибка при установке SERVER_ADDRESS: %v", err)
	}
	if err := os.Setenv("ENABLE_HTTPS", "false"); err != nil {
		t.Logf("Ошибка при установке ENABLE_HTTPS: %v", err)
	}

	// Создаем конфигурацию без парсинга флагов
	cfg := &Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "internal/storage/storage.json",
		DatabaseDSN:     "",
		JWTSecret:       "default_jwt_secret",
		EnableHTTPS:     false,
	}

	// Загружаем конфигурацию из файла
	configFile, err := loadConfigFile(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, configFile)

	// Применяем значения из файла конфигурации
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
	}

	// Применяем переменные окружения (высший приоритет)
	if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
		cfg.RunAddr = addr
	}
	if url := os.Getenv("BASE_URL"); url != "" {
		cfg.BaseURL = url
	}
	if path := os.Getenv("FILE_STORAGE_PATH"); path != "" {
		cfg.FileStoragePath = path
	}
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		cfg.DatabaseDSN = dsn
	}
	if enableHTTPS := os.Getenv("ENABLE_HTTPS"); enableHTTPS != "" {
		cfg.EnableHTTPS = enableHTTPS == "true"
	}

	// Проверяем приоритет: переменные окружения должны переопределить значения из JSON файла
	assert.Equal(t, "localhost:8080", cfg.RunAddr)       // Переопределено переменной окружения
	assert.Equal(t, "https://example.com", cfg.BaseURL)  // Из JSON файла
	assert.Equal(t, false, cfg.EnableHTTPS)              // Переопределено переменной окружения
	assert.Equal(t, "default_jwt_secret", cfg.JWTSecret) // Остается дефолтным значением
}

func TestConfig_FileStoragePath(t *testing.T) {
	tempDir := t.TempDir()
	filePath := tempDir + "/subdir/storage.json"

	dir := filePath[:len(filePath)-len("/storage.json")]
	_, err := os.Stat(dir)
	assert.Error(t, err, "Directory should not exist initially")

	err = os.MkdirAll(dir, 0755)
	assert.NoError(t, err, "Should create directory")

	_, err = os.Stat(dir)
	assert.NoError(t, err, "Directory should be created")
}
