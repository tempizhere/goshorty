package service

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// mockRepository для тестов
type mockRepository struct {
	store map[string]string
}

func (m *mockRepository) Save(id, url string) error {
	if id == "fail" {
		return errors.New("save failed")
	}
	m.store[id] = url
	return nil
}

func (m *mockRepository) Get(id string) (string, bool) {
	url, exists := m.store[id]
	return url, exists
}

func (m *mockRepository) Clear() {
	m.store = make(map[string]string)
}

func TestService(t *testing.T) {
	repo := &mockRepository{store: make(map[string]string)}
	svc := NewService(repo, "http://localhost:8080")

	// Тест CreateShortURL
	shortURL, err := svc.CreateShortURL("https://example.com")
	assert.NoError(t, err, "CreateShortURL should not return error")
	assert.True(t, strings.HasPrefix(shortURL, "http://localhost:8080/"), "Short URL should start with baseURL")
	id := svc.ExtractIDFromShortURL(shortURL)
	assert.Len(t, id, 8, "ID should be 8 characters long")

	// Тест CreateShortURL с пустым URL
	_, err = svc.CreateShortURL("")
	assert.EqualError(t, err, "empty URL", "CreateShortURL should return empty URL error")

	// Тест CreateShortURLWithID с ошибкой сохранения
	_, err = svc.CreateShortURLWithID("https://fail.com", "fail")
	assert.EqualError(t, err, "save failed", "CreateShortURLWithID should return save error")

	// Тест CreateShortURLWithID с существующим ID
	err = repo.Save("id1", "https://example.com")
	assert.NoError(t, err, "Save should not return error")
	_, err = svc.CreateShortURLWithID("https://another.com", "id1")
	assert.ErrorIs(t, err, ErrIDAlreadyExists, "CreateShortURLWithID should return ID already exists error")

	// Тест GetOriginalURL
	url, exists := svc.GetOriginalURL(id)
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест GetOriginalURL для несуществующего ID
	_, exists = svc.GetOriginalURL("unknown")
	assert.False(t, exists, "URL should not exist")

	// Тест ExtractIDFromShortURL
	extractedID := svc.ExtractIDFromShortURL("http://localhost:8080/abcdef12")
	assert.Equal(t, "abcdef12", extractedID, "Extracted ID should match")
}
