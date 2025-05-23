package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestFileRepository(t *testing.T) {
	// Создаём временную директорию для теста
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage.json")

	// Создаём репозиторий
	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")

	// Тест 1: Сохранение и получение URL
	shortID, err := repo.Save("testID", "https://example.com")
	assert.NoError(t, err, "Failed to save URL")
	assert.Equal(t, "testID", shortID, "Returned short_id should match")
	url, exists := repo.Get("testID")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 2: Сохранение существующего URL
	existingID, err := repo.Save("newID", "https://example.com")
	assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")
	assert.Equal(t, "testID", existingID, "Should return existing short_id")
	url, exists = repo.Get("testID")
	assert.True(t, exists, "Original URL should still exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 3: Восстановление данных
	repo2, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create second file repository")
	url, exists = repo2.Get("testID")
	assert.True(t, exists, "URL should be restored")
	assert.Equal(t, "https://example.com", url, "Restored URL mismatch")

	// Тест 4: Очистка хранилища
	repo.Clear()
	_, exists = repo.Get("testID")
	assert.False(t, exists, "URL should be cleared")
	_, err = os.Stat(tempFile)
	assert.NoError(t, err, "File should exist after clear")

	// Тест 5: Обработка некорректного JSON
	err = os.WriteFile(tempFile, []byte("invalid json\n"), 0644)
	assert.NoError(t, err, "Failed to write invalid JSON")
	repo3, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Should handle invalid JSON lines")
	_, exists = repo3.Get("testID")
	assert.False(t, exists, "No URLs should be loaded from invalid JSON")
}

func TestFileRepository_NonExistentDir(t *testing.T) {
	// Создаём временную директорию для теста
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "subdir/storage.json")

	// Создаём репозиторий
	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create repository in non-existent dir")

	// Тест 6: Сохранение URL
	_, err = repo.Save("testID", "https://example.com")
	assert.NoError(t, err, "Failed to save URL in new dir")
}

func TestFileRepository_FilePermissionError(t *testing.T) {
	// Создаём временную директорию для теста
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage.json")

	// Создаём файл с правами только на чтение
	err := os.WriteFile(tempFile, []byte{}, 0400)
	assert.NoError(t, err, "Failed to create read-only file")

	// Создаём репозиторий
	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Should create repository despite read-only file")

	// Тест 7: Попытка сохранения (должна пройти, так как файл пересоздаётся)
	_, err = repo.Save("testID", "https://example.com")
	assert.NoError(t, err, "Failed to save URL to read-only file")
}
