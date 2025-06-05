package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/models"
	"go.uber.org/zap"
)

func TestFileRepository(t *testing.T) {
	userID := "test_user"

	// Тест 1: Сохранение и получение URL
	t.Run("Save and Get URL", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "storage.json")
		repo, err := NewFileRepository(tempFile, zap.NewNop())
		assert.NoError(t, err, "Failed to create file repository")

		shortID, err := repo.Save("testID", "https://example.com", userID)
		assert.NoError(t, err, "Failed to save URL")
		assert.Equal(t, "testID", shortID, "Returned short_id should match")
		url, exists := repo.Get("testID")
		assert.True(t, exists, "URL should exist")
		assert.Equal(t, "https://example.com", url, "URL should match")
	})

	// Тест 2: Сохранение существующего URL
	t.Run("Save Duplicate URL", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "storage.json")
		repo, err := NewFileRepository(tempFile, zap.NewNop())
		assert.NoError(t, err, "Failed to create file repository")

		_, err = repo.Save("testID", "https://example.com", userID)
		assert.NoError(t, err, "Failed to save URL")
		existingID, err := repo.Save("newID", "https://example.com", userID)
		assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")
		assert.Equal(t, "testID", existingID, "Should return existing short_id")
		url, exists := repo.Get("testID")
		assert.True(t, exists, "Original URL should still exist")
		assert.Equal(t, "https://example.com", url, "URL should match")
	})

	// Тест 3: Восстановление данных и получение URL по UserID
	t.Run("Restore Data and Get URLs by UserID", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "storage.json")
		repo, err := NewFileRepository(tempFile, zap.NewNop())
		assert.NoError(t, err, "Failed to create file repository")

		_, err = repo.Save("testID", "https://example.com", userID)
		assert.NoError(t, err, "Failed to save URL")

		repo2, err := NewFileRepository(tempFile, zap.NewNop())
		assert.NoError(t, err, "Failed to create second file repository")
		url, exists := repo2.Get("testID")
		assert.True(t, exists, "URL should be restored")
		assert.Equal(t, "https://example.com", url, "Restored URL mismatch")

		urls, err := repo2.GetURLsByUserID(userID)
		assert.NoError(t, err, "Failed to get URLs by userID")
		assert.Len(t, urls, 1, "Expected one URL")
		assert.Equal(t, models.URL{ShortID: "testID", OriginalURL: "https://example.com", UserID: userID}, urls[0], "URL should match")
		urls, err = repo2.GetURLsByUserID("unknown_user")
		assert.NoError(t, err, "Failed to get URLs by unknown userID")
		assert.Len(t, urls, 0, "Expected no URLs for unknown user")
	})

	// Тест 4: Очистка хранилища
	t.Run("Clear Storage", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "storage.json")
		repo, err := NewFileRepository(tempFile, zap.NewNop())
		assert.NoError(t, err, "Failed to create file repository")

		_, err = repo.Save("testID", "https://example.com", userID)
		assert.NoError(t, err, "Failed to save URL")
		repo.Clear()
		_, exists := repo.Get("testID")
		assert.False(t, exists, "URL should be cleared")
		_, err = os.Stat(tempFile)
		assert.NoError(t, err, "File should exist after clear")
	})

	// Тест 5: Обработка некорректного JSON
	t.Run("Handle Invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "storage.json")
		err := os.WriteFile(tempFile, []byte("invalid json\n"), 0644)
		assert.NoError(t, err, "Failed to write invalid JSON")
		repo3, err := NewFileRepository(tempFile, zap.NewNop())
		assert.NoError(t, err, "Should handle invalid JSON lines")
		_, exists := repo3.Get("testID")
		assert.False(t, exists, "No URLs should be loaded from invalid JSON")
	})
}

func TestFileRepository_NonExistentDir(t *testing.T) {
	// Создаём временную директорию для теста
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "subdir/storage.json")
	userID := "test_user"

	// Создаём репозиторий
	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create repository in non-existent dir")

	// Тест 6: Сохранение URL
	_, err = repo.Save("testID", "https://example.com", userID)
	assert.NoError(t, err, "Failed to save URL in new dir")
}

func TestFileRepository_FilePermissionError(t *testing.T) {
	// Создаём временную директорию для теста
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage.json")
	userID := "test_user"

	// Создаём файл с правами только на чтение
	err := os.WriteFile(tempFile, []byte{}, 0400)
	assert.NoError(t, err, "Failed to create read-only file")

	// Создаём репозиторий
	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Should create repository despite read-only file")

	// Тест 7: Попытка сохранения (должна пройти, так как файл пересоздаётся)
	_, err = repo.Save("testID", "https://example.com", userID)
	assert.NoError(t, err, "Failed to save URL to read-only file")
}
