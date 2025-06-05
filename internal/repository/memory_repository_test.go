package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/models"
)

func TestMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()
	userID := "test_user"

	// Проверяем, что MemoryRepository реализует интерфейс Repository
	var _ Repository = (*MemoryRepository)(nil)

	// Тест 1: Сохранение и получение URL
	shortID, err := repo.Save("id1", "https://example.com", userID)
	assert.NoError(t, err, "Save should not return error")
	assert.Equal(t, "id1", shortID, "Returned short_id should match")
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 2: Сохранение существующего URL
	existingID, err := repo.Save("id2", "https://example.com", userID)
	assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")
	assert.Equal(t, "id1", existingID, "Should return existing short_id")
	url, exists = repo.Get("id1")
	assert.True(t, exists, "Original URL should still exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 3: Перезапись существующего ID
	_, err = repo.Save("id1", "https://new-example.com", userID)
	assert.NoError(t, err, "Save should not return error for overwrite")
	url, exists = repo.Get("id1")
	assert.True(t, exists, "URL should still exist")
	assert.Equal(t, "https://new-example.com", url, "URL should be updated")

	// Тест 4: Получение несуществующего ID
	_, exists = repo.Get("id3")
	assert.False(t, exists, "URL should not exist")

	// Тест 5: Очистка хранилища
	repo.Clear()
	_, exists = repo.Get("id1")
	assert.False(t, exists, "URL should be cleared")

	// Тест 6: Получение URL по UserID
	_, err = repo.Save("id1", "https://example.com", userID)
	assert.NoError(t, err, "Failed to save URL")
	urls, err := repo.GetURLsByUserID(userID)
	assert.NoError(t, err, "Failed to get URLs by userID")
	assert.Len(t, urls, 1, "Expected one URL")
	assert.Equal(t, models.URL{ShortID: "id1", OriginalURL: "https://example.com", UserID: userID}, urls[0], "URL should match")
	urls, err = repo.GetURLsByUserID("unknown_user")
	assert.NoError(t, err, "Failed to get URLs by unknown userID")
	assert.Len(t, urls, 0, "Expected no URLs for unknown user")
}
