package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/models"
)

func TestMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()

	// Проверяем, что MemoryRepository реализует интерфейс Repository
	var _ Repository = (*MemoryRepository)(nil)

	// Тест 1: Сохранение и получение URL
	shortID, err := repo.Save("id1", "https://example.com", "user1")
	assert.NoError(t, err, "Save should not return error")
	assert.Equal(t, "id1", shortID, "Returned short_id should match")
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url.OriginalURL, "URL should match")

	// Тест 2: Сохранение существующего URL
	existingID, err := repo.Save("id2", "https://example.com", "user1")
	assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")
	assert.Equal(t, "id1", existingID, "Should return existing short_id")
	url, exists = repo.Get("id1")
	assert.True(t, exists, "Original URL should still exist")
	assert.Equal(t, "https://example.com", url.OriginalURL, "URL should match")

	// Тест 3: Перезапись существующего ID
	_, err = repo.Save("id1", "https://new-example.com", "user1")
	assert.NoError(t, err, "Save should not return error for overwrite")
	url, exists = repo.Get("id1")
	assert.True(t, exists, "URL should still exist")
	assert.Equal(t, "https://new-example.com", url.OriginalURL, "URL should be updated")

	// Тест 4: Получение несуществующего ID
	url, exists = repo.Get("id3")
	assert.False(t, exists, "URL should not exist")
	assert.Equal(t, models.URL{}, url, "Should return empty URL struct")

	// Тест 5: Очистка хранилища
	repo.Clear()
	_, exists = repo.Get("id1")
	assert.False(t, exists, "URL should be cleared")
}

func TestMemoryRepository_BatchSave(t *testing.T) {
	repo := NewMemoryRepository()

	// Тест 1: Успешное пакетное сохранение
	urls := map[string]string{
		"id1": "https://example1.com",
		"id2": "https://example2.com",
		"id3": "https://example3.com",
	}
	err := repo.BatchSave(urls, "user1")
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
}

func TestMemoryRepository_GetURLsByUserID(t *testing.T) {
	repo := NewMemoryRepository()

	// Сохраняем URL для разных пользователей
	_, err := repo.Save("id1", "https://example1.com", "user1")
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

func TestMemoryRepository_BatchDelete(t *testing.T) {
	repo := NewMemoryRepository()

	// Сохраняем URL для пользователя
	_, err := repo.Save("id1", "https://example1.com", "user1")
	assert.NoError(t, err)
	_, err = repo.Save("id2", "https://example2.com", "user1")
	assert.NoError(t, err)
	_, err = repo.Save("id3", "https://example3.com", "user1")
	assert.NoError(t, err)
	_, err = repo.Save("id4", "https://example4.com", "user2")
	assert.NoError(t, err)

	// Тест 1: Успешное пакетное удаление
	err = repo.BatchDelete("user1", []string{"id1", "id2"})
	assert.NoError(t, err, "BatchDelete should succeed")

	// Проверяем, что URL помечены как удалённые
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should still exist")
	assert.True(t, url.DeletedFlag, "URL should be marked as deleted")

	url, exists = repo.Get("id2")
	assert.True(t, exists, "URL should still exist")
	assert.True(t, url.DeletedFlag, "URL should be marked as deleted")

	// Проверяем, что id3 не затронут
	url, exists = repo.Get("id3")
	assert.True(t, exists, "URL should still exist")
	assert.False(t, url.DeletedFlag, "URL should not be marked as deleted")

	// Проверяем, что URL другого пользователя не затронут
	url, exists = repo.Get("id4")
	assert.True(t, exists, "URL should still exist")
	assert.False(t, url.DeletedFlag, "URL should not be marked as deleted")

	// Тест 2: Удаление несуществующих ID
	err = repo.BatchDelete("user1", []string{"nonexistent1", "nonexistent2"})
	assert.NoError(t, err, "BatchDelete should succeed for non-existent IDs")

	// Тест 3: Удаление ID другого пользователя
	err = repo.BatchDelete("user1", []string{"id4"})
	assert.NoError(t, err, "BatchDelete should succeed")

	// Проверяем, что URL другого пользователя не затронут
	url, exists = repo.Get("id4")
	assert.True(t, exists, "URL should still exist")
	assert.False(t, url.DeletedFlag, "URL should not be marked as deleted")
}
