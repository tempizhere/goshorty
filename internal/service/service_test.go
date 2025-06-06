package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
)

// mockRepository для тестов
type mockRepository struct {
	store map[string]models.URL
}

func (m *mockRepository) Save(id, url, userID string) (string, error) {
	if id == "fail" {
		return "", errors.New("save failed")
	}
	for existingID, existingURL := range m.store {
		if existingURL.OriginalURL == url {
			return existingID, repository.ErrURLExists
		}
	}
	m.store[id] = models.URL{
		ShortID:     id,
		OriginalURL: url,
		UserID:      userID,
		DeletedFlag: false,
	}
	return id, nil
}

func (m *mockRepository) Get(id string) (models.URL, bool) {
	url, exists := m.store[id]
	return url, exists
}

func (m *mockRepository) Clear() {
	m.store = make(map[string]models.URL)
}

func (m *mockRepository) BatchSave(urls map[string]string, userID string) error {
	for _, url := range urls {
		if url == "https://fail.com" {
			return errors.New("batch save failed")
		}
	}
	for id, url := range urls {
		for _, existingURL := range m.store {
			if existingURL.OriginalURL == url {
				return repository.ErrURLExists
			}
		}
		m.store[id] = models.URL{
			ShortID:     id,
			OriginalURL: url,
			UserID:      userID,
			DeletedFlag: false,
		}
	}
	return nil
}

func (m *mockRepository) GetURLsByUserID(userID string) ([]models.URL, error) {
	var urls []models.URL
	for _, u := range m.store {
		if u.UserID == userID {
			urls = append(urls, u)
		}
	}
	return urls, nil
}

func (m *mockRepository) BatchDelete(userID string, ids []string) error {
	for _, id := range ids {
		if u, exists := m.store[id]; exists && u.UserID == userID {
			u.DeletedFlag = true
			m.store[id] = u
		}
	}
	return nil
}

func TestService(t *testing.T) {
	const testUserID = "test_user"
	repo := &mockRepository{store: make(map[string]models.URL)}
	svc := NewService(repo, "http://localhost:8080", "secret")

	// Тест 1: CreateShortURL успех
	shortURL, err := svc.CreateShortURL("https://example.com", testUserID)
	assert.NoError(t, err, "CreateShortURL should not return error")
	assert.True(t, strings.HasPrefix(shortURL, "http://localhost:8080/"), "Short URL should start with baseURL")
	id := svc.ExtractIDFromShortURL(shortURL)
	assert.Len(t, id, 8, "ID should be 8 characters long")

	// Тест 2: CreateShortURL с дублирующимся URL
	duplicateURL, err := svc.CreateShortURL("https://example.com", testUserID)
	assert.ErrorIs(t, err, repository.ErrURLExists, "CreateShortURL should return ErrURLExists for duplicate URL")
	assert.Equal(t, shortURL, duplicateURL, "Should return existing short URL")

	// Тест 3: CreateShortURL с пустым URL
	_, err = svc.CreateShortURL("", testUserID)
	assert.EqualError(t, err, "empty URL", "CreateShortURL should return empty URL error")

	// Тест 4: CreateShortURLWithID с ошибкой сохранения
	_, err = svc.CreateShortURLWithID("https://fail.com", "fail", testUserID)
	assert.EqualError(t, err, "save failed", "CreateShortURLWithID should return save error")

	// Тест 5: CreateShortURLWithID с существующим ID
	_, err = svc.CreateShortURLWithID("https://another.com", id, testUserID)
	assert.ErrorIs(t, err, ErrIDAlreadyExists, "CreateShortURLWithID should return ID already exists error")

	// Тест 6: CreateShortURLWithID с дублирующимся URL
	_, err = repo.Save("existingID", "https://another.com", testUserID)
	assert.NoError(t, err, "Save should not return error")
	duplicateShortURL, err := svc.CreateShortURLWithID("https://another.com", "newID", testUserID)
	assert.ErrorIs(t, err, repository.ErrURLExists, "CreateShortURLWithID should return ErrURLExists for duplicate URL")
	assert.Equal(t, "http://localhost:8080/existingID", duplicateShortURL, "Should return existing short URL")

	// Тест 7: GetOriginalURL
	url, exists := svc.GetOriginalURL(id)
	assert.True(t, exists, "URL should exist")
	assert.Equal(t, "https://example.com", url, "URL should match")

	// Тест 8: GetOriginalURL для несуществующего ID
	_, exists = svc.GetOriginalURL("unknown")
	assert.False(t, exists, "URL should not exist")

	// Тест 9: ExtractIDFromShortURL
	extractedID := svc.ExtractIDFromShortURL("http://localhost:8080/abcdef12")
	assert.Equal(t, "abcdef12", extractedID, "Extracted ID should match")

	// Тест 10: GetURLsByUserID успех
	urls, err := svc.GetURLsByUserID(testUserID)
	assert.NoError(t, err, "GetURLsByUserID should not return error")
	assert.Len(t, urls, 2, "Should return two URLs for test user")
	assert.Equal(t, "https://example.com", urls[0].OriginalURL, "First URL should match")
	assert.Equal(t, "https://another.com", urls[1].OriginalURL, "Second URL should match")

	// Тест 11: GetURLsByUserID для несуществующего пользователя
	urls, err = svc.GetURLsByUserID("unknown_user")
	assert.NoError(t, err, "GetURLsByUserID should not return error")
	assert.Len(t, urls, 0, "Should return empty list for unknown user")

	// Тест 12: BatchDelete успех
	err = svc.BatchDelete(testUserID, []string{id, "existingID"})
	assert.NoError(t, err, "BatchDelete should not return error")
	u, exists := repo.Get(id)
	assert.True(t, exists, "URL should still exist")
	assert.True(t, u.DeletedFlag, "URL should be marked as deleted")
	_, exists = svc.GetOriginalURL(id)
	assert.False(t, exists, "GetOriginalURL should return false for deleted URL")

	// Тест 13: BatchDelete для несуществующих ID
	err = svc.BatchDelete(testUserID, []string{"unknown"})
	assert.NoError(t, err, "BatchDelete should not return error")
}
