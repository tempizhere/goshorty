package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()

	// Проверяем, что MemoryRepository реализует интерфейс Repository
	var _ Repository = (*MemoryRepository)(nil)

	// Тест 1: Сохранение и получение URL
	shortID, err := repo.Save("id1", "https://example.com")
	assert.NoError(t, err, "Save should not return error")
	assert.Equal(t, "id1", shortID, "Returned short_id should match")
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 2: Сохранение существующего URL
	existingID, err := repo.Save("id2", "https://example.com")
	assert.ErrorIs(t, err, ErrURLExists, "Expected ErrURLExists for duplicate URL")
	assert.Equal(t, "id1", existingID, "Should return existing short_id")
	url, exists = repo.Get("id1")
	assert.True(t, exists, "Original URL should still exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 3: Перезапись существующего ID
	_, err = repo.Save("id1", "https://new-example.com")
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
}
