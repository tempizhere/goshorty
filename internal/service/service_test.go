package service

import (
	"errors"
	"strings"
	"testing"
	"time"

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
		if u.UserID == userID && !u.DeletedFlag {
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

	// Проверяем, что URLs содержат ожидаемые значения в любом порядке
	var foundExample, foundAnother bool
	for _, u := range urls {
		if u.OriginalURL == "https://example.com" {
			foundExample = true
			assert.True(t, strings.HasPrefix(u.ShortURL, "http://localhost:8080/"), "Short URL should start with baseURL")
		}
		if u.OriginalURL == "https://another.com" {
			foundAnother = true
		}
	}
	assert.True(t, foundExample, "Should contain https://example.com")
	assert.True(t, foundAnother, "Should contain https://another.com")

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

	// Тест 14: BatchDeleteAsync успех
	repo = &mockRepository{store: make(map[string]models.URL)}
	svc = NewService(repo, "http://localhost:8080", "secret")
	_, err = repo.Save("testID", "https://test.com", testUserID)
	assert.NoError(t, err, "Save should not return error")
	svc.BatchDeleteAsync(testUserID, []string{"testID"})
	// Поскольку асинхронный вызов, проверяем через небольшой таймер
	time.Sleep(100 * time.Millisecond)
	u, exists = repo.Get("testID")
	assert.True(t, exists, "URL should still exist")
	assert.True(t, u.DeletedFlag, "URL should be marked as deleted")
}

func TestBatchShorten(t *testing.T) {
	const testUserID = "test_user"
	tests := []struct {
		name    string
		reqs    []models.BatchRequest
		wantErr string
		wantLen int
	}{
		{
			name:    "empty batch",
			reqs:    []models.BatchRequest{},
			wantErr: ErrEmptyBatch.Error(),
			wantLen: 0,
		},
		{
			name: "duplicate correlation_id",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "1", OriginalURL: "https://example.org"},
			},
			wantErr: ErrDuplicateCorrID.Error(),
			wantLen: 0,
		},
		{
			name: "empty URL",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: ""},
			},
			wantErr: ErrEmptyURL.Error(),
			wantLen: 0,
		},
		{
			name: "successful batch",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "2", OriginalURL: "https://example.org"},
			},
			wantErr: "",
			wantLen: 2,
		},
		{
			name: "batch save failure",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://fail.com"},
				{CorrelationID: "2", OriginalURL: "https://example.org"},
			},
			wantErr: "batch save failed",
			wantLen: 0,
		},
		{
			name: "duplicate URL",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://duplicate.com"},
				{CorrelationID: "2", OriginalURL: "https://duplicate.com"},
			},
			wantErr: repository.ErrURLExists.Error(),
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{store: make(map[string]models.URL)}
			svc := NewService(repo, "http://localhost:8080", "secret")

			// Подготовка данных для теста "duplicate URL"
			if tt.name == "duplicate URL" {
				_, err := repo.Save("existingID", "https://duplicate.com", testUserID)
				assert.NoError(t, err, "Save should not return error")
			}

			resp, err := svc.BatchShorten(tt.reqs, testUserID)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				if tt.wantLen == 0 {
					assert.Nil(t, resp)
				} else {
					assert.Len(t, resp, tt.wantLen)
				}
				return
			}

			assert.NoError(t, err)
			assert.Len(t, resp, tt.wantLen)

			// Проверяем корректность созданных URL
			for i, r := range resp {
				// Проверяем формат короткого URL
				assert.True(t, strings.HasPrefix(r.ShortURL, "http://localhost:8080/"))
				assert.Equal(t, tt.reqs[i].CorrelationID, r.CorrelationID)

				// Проверяем, что URL сохранен в хранилище
				id := svc.ExtractIDFromShortURL(r.ShortURL)
				originalURL, exists := repo.Get(id)
				assert.True(t, exists)
				assert.Equal(t, tt.reqs[i].OriginalURL, originalURL.OriginalURL)
			}
		})
	}
}

func TestJWT(t *testing.T) {
	svc := NewService(&mockRepository{store: make(map[string]models.URL)}, "http://localhost:8080", "secret")

	// Тест 1: GenerateUserID успех
	userID, err := svc.GenerateUserID()
	assert.NoError(t, err, "GenerateUserID should not return error")
	assert.Len(t, userID, 8, "UserID should be 8 characters long")

	// Тест 2: GenerateJWT и ParseJWT успех
	token, err := svc.GenerateJWT(userID)
	assert.NoError(t, err, "GenerateJWT should not return error")
	parsedUserID, err := svc.ParseJWT(token)
	assert.NoError(t, err, "ParseJWT should not return error")
	assert.Equal(t, userID, parsedUserID, "Parsed UserID should match")

	// Тест 3: ParseJWT с некорректным токеном
	_, err = svc.ParseJWT("invalid.token")
	assert.ErrorIs(t, err, ErrInvalidToken, "ParseJWT should return ErrInvalidToken")

	// Тест 4: ParseJWT с неверным секретом
	svcWrongSecret := NewService(&mockRepository{store: make(map[string]models.URL)}, "http://localhost:8080", "wrong_secret")
	_, err = svcWrongSecret.ParseJWT(token)
	assert.ErrorIs(t, err, ErrInvalidToken, "ParseJWT should return ErrInvalidToken with wrong secret")
}
