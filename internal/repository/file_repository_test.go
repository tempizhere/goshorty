package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestFileRepository тестирует основные операции FileRepository
func TestFileRepository(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage.json")

	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	shortID, err := repo.Save("testID", "https://example.com", "user1")
	assert.NoError(t, err, "Failed to save URL")
	assert.Equal(t, "testID", shortID, "Returned short_id should match")
	url, exists := repo.Get("testID")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url.OriginalURL, "URL should match")

	// Тест 2: Сохранение существующего URL
	existingID, err := repo.Save("newID", "https://example.com", "user1")
	assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")
	assert.Equal(t, "testID", existingID, "Should return existing short_id")
	url, exists = repo.Get("testID")
	assert.True(t, exists, "Original URL should still exist")
	assert.Equal(t, "https://example.com", url.OriginalURL, "URL should match")

	// Тест 3: Восстановление данных
	repo2, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create second file repository")
	url, exists = repo2.Get("testID")
	assert.True(t, exists, "URL should be restored")
	assert.Equal(t, "https://example.com", url.OriginalURL, "Restored URL mismatch")

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

// TestFileRepository_NonExistentDir тестирует создание репозитория в несуществующей директории
func TestFileRepository_NonExistentDir(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "subdir/storage.json")

	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create repository in non-existent dir")
	_, err = repo.Save("testID", "https://example.com", "user1")
	assert.NoError(t, err, "Failed to save URL in new dir")
}

// TestFileRepository_FilePermissionError тестирует обработку ошибок прав доступа к файлу
func TestFileRepository_FilePermissionError(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage.json")

	err := os.WriteFile(tempFile, []byte{}, 0400)
	assert.NoError(t, err, "Failed to create read-only file")

	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Should create repository despite read-only file")
	_, err = repo.Save("testID", "https://example.com", "user1")
	assert.NoError(t, err, "Failed to save URL to read-only file")
}

// TestFileRepository_BatchSave тестирует пакетное сохранение URL
func TestFileRepository_BatchSave(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage_batch.json")

	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	urls := map[string]string{
		"id1": "https://example1.com",
		"id2": "https://example2.com",
		"id3": "https://example3.com",
	}
	err = repo.BatchSave(urls, "user1")
	assert.NoError(t, err, "BatchSave should succeed")

	// Проверяем, что все URL сохранены
	for id, expectedURL := range urls {
		url, exists := repo.Get(id)
		assert.True(t, exists, "URL should exist")
		assert.Equal(t, expectedURL, url.OriginalURL, "URL should match")
		assert.Equal(t, "user1", url.UserID, "UserID should match")
	}

	// Тест 2: Попытка пакетного сохранения с дублирующимся URL
	duplicateURLs := map[string]string{
		"id4": "https://example1.com", // Дублирующийся URL
		"id5": "https://example4.com",
	}
	err = repo.BatchSave(duplicateURLs, "user1")
	assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")

	// Проверяем, что новые URL не были добавлены
	_, exists := repo.Get("id4")
	assert.False(t, exists, "Duplicate URL should not be saved")
	_, exists = repo.Get("id5")
	assert.False(t, exists, "URL after duplicate should not be saved")
}

func TestFileRepository_GetURLsByUserID(t *testing.T) {
	// Создаём временную директорию для теста
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage_user.json")

	// Создаём репозиторий
	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")

	// Сохраняем URL для разных пользователей
	_, err = repo.Save("id1", "https://example1.com", "user1")
	assert.NoError(t, err)
	_, err = repo.Save("id2", "https://example2.com", "user1")
	assert.NoError(t, err)
	_, err = repo.Save("id3", "https://example3.com", "user2")
	assert.NoError(t, err)

	// Тест 1: Получение URL для user1
	urls, err := repo.GetURLsByUserID("user1")
	assert.NoError(t, err, "GetURLsByUserID should succeed")
	assert.Len(t, urls, 2, "Should return 2 URLs for user1")

	// Проверяем содержимое
	urlMap := make(map[string]string)
	for _, url := range urls {
		urlMap[url.ShortID] = url.OriginalURL
	}
	assert.Equal(t, "https://example1.com", urlMap["id1"])
	assert.Equal(t, "https://example2.com", urlMap["id2"])

	// Тест 2: Получение URL для user2
	urls, err = repo.GetURLsByUserID("user2")
	assert.NoError(t, err, "GetURLsByUserID should succeed")
	assert.Len(t, urls, 1, "Should return 1 URL for user2")
	assert.Equal(t, "https://example3.com", urls[0].OriginalURL)

	// Тест 3: Получение URL для несуществующего пользователя
	urls, err = repo.GetURLsByUserID("nonexistent")
	assert.NoError(t, err, "GetURLsByUserID should succeed for non-existent user")
	assert.Len(t, urls, 0, "Should return empty slice for non-existent user")
}

func TestFileRepository_Close(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "storage_close.json")

	repo, err := NewFileRepository(tempFile, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")

	// Сохраняем несколько URL
	_, err = repo.Save("id1", "https://example1.com", "user1")
	assert.NoError(t, err)
	_, err = repo.Save("id2", "https://example2.com", "user1")
	assert.NoError(t, err)

	// Тест: Close должен завершаться без ошибок
	err = repo.Close()
	assert.NoError(t, err, "Close should not return error")

	// Проверяем, что данные все еще доступны после Close
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should still exist after Close")
	assert.Equal(t, "https://example1.com", url.OriginalURL)

	// Проверяем, что файл все еще существует
	_, err = os.Stat(tempFile)
	assert.NoError(t, err, "File should still exist after Close")
}
