package repository

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()

	// Проверяем, что MemoryRepository реализует интерфейс Repository
	var _ Repository = (*MemoryRepository)(nil)

	// Тест 1: Сохранение и получение URL
	err := repo.Save("id1", "https://example.com")
	assert.NoError(t, err, "Save should not return error")
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 2: Перезапись существующего ID
	err = repo.Save("id1", "https://new-example.com")
	assert.NoError(t, err, "Save should not return error for overwrite")
	url, exists = repo.Get("id1")
	assert.True(t, exists, "URL should still exist")
	assert.Equal(t, "https://new-example.com", url, "URL should be updated")

	// Тест 3: Получение несуществующего ID
	_, exists = repo.Get("id2")
	assert.False(t, exists, "URL should not exist")

	// Тест 4: Очистка хранилища
	repo.Clear()
	_, exists = repo.Get("id1")
	assert.False(t, exists, "URL should be cleared")
}
