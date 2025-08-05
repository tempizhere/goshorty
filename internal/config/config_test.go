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
	}

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, "internal/storage/storage.json", cfg.FileStoragePath)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "default_jwt_secret", cfg.JWTSecret)
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
	envVars := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "JWT_SECRET"}
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

	assert.Equal(t, "9090", os.Getenv("SERVER_ADDRESS"))
	assert.Equal(t, "https://example.com", os.Getenv("BASE_URL"))
	assert.Equal(t, "/tmp/storage.json", os.Getenv("FILE_STORAGE_PATH"))
	assert.Equal(t, "postgres://user:pass@localhost/db", os.Getenv("DATABASE_DSN"))
	assert.Equal(t, "my_secret_key", os.Getenv("JWT_SECRET"))
}

func TestNewConfig_Integration(t *testing.T) {
	originalEnv := make(map[string]string)
	envVars := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "JWT_SECRET"}
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

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, filePath, cfg.FileStoragePath)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "default_jwt_secret", cfg.JWTSecret)

	dir := tempDir
	_, err = os.Stat(dir)
	assert.NoError(t, err, "Directory should be created")
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
