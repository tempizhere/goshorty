package repository

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()

	// Тест Save и Get
	err := repo.Save("id1", "https://example.com")
	assert.NoError(t, err, "Save should not return error")
	url, exists := repo.Get("id1")
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест Get для несуществующего ID
	_, exists = repo.Get("id2")
	assert.False(t, exists, "URL should not exist")

	// Тест Clear
	repo.Clear()
	_, exists = repo.Get("id1")
	assert.False(t, exists, "URL should be cleared")
}
