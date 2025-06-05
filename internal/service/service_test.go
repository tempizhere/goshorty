package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
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
	m.store[id] = models.URL{ShortID: id, OriginalURL: url, UserID: userID}
	return id, nil
}

func (m *mockRepository) Get(id string) (string, bool) {
	u, exists := m.store[id]
	return u.OriginalURL, exists
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
		m.store[id] = models.URL{ShortID: id, OriginalURL: url, UserID: userID}
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

func TestService(t *testing.T) {
	repo := &mockRepository{store: make(map[string]models.URL)}
	svc := NewService(repo, "http://localhost:8080", "test_secret")

	// Тест 1: CreateShortURL успех
	userID := "test_user"
	shortURL, err := svc.CreateShortURL("https://example.com", userID)
	assert.NoError(t, err, "CreateShortURL should not return error")
	assert.True(t, strings.HasPrefix(shortURL, "http://localhost:8080/"), "Short URL should start with baseURL")
	id := svc.ExtractIDFromShortURL(shortURL)
	assert.Len(t, id, 8, "ID should be 8 characters long")

	// Тест 2: CreateShortURL с дублирующимся URL
	duplicateURL, err := svc.CreateShortURL("https://example.com", userID)
	assert.ErrorIs(t, err, repository.ErrURLExists, "CreateShortURL should return ErrURLExists for duplicate URL")
	assert.Equal(t, shortURL, duplicateURL, "Should return existing short URL")

	// Тест 3: CreateShortURL с пустым URL
	_, err = svc.CreateShortURL("", userID)
	assert.EqualError(t, err, "empty URL", "CreateShortURL should return empty URL error")

	// Тест 4: CreateShortURLWithID с ошибкой сохранения
	_, err = svc.CreateShortURLWithID("https://fail.com", "fail", userID)
	assert.EqualError(t, err, "save failed", "CreateShortURLWithID should return save error")

	// Тест 5: CreateShortURLWithID с существующим ID
	_, err = svc.CreateShortURLWithID("https://another.com", id, userID)
	assert.ErrorIs(t, err, ErrIDAlreadyExists, "CreateShortURLWithID should return ID already exists error")

	// Тест 6: CreateShortURLWithID с дублирующимся URL
	_, err = repo.Save("existingID", "https://another.com", userID)
	assert.NoError(t, err, "Save should not return error")
	duplicateShortURL, err := svc.CreateShortURLWithID("https://another.com", "newID", userID)
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
}

func TestBatchShorten(t *testing.T) {
	tests := []struct {
		name    string
		reqs    []models.BatchRequest
		userID  string
		wantErr string
		wantLen int
	}{
		{
			name:    "empty batch",
			reqs:    []models.BatchRequest{},
			userID:  "test_user",
			wantErr: ErrEmptyBatch.Error(),
			wantLen: 0,
		},
		{
			name: "duplicate correlation_id",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "1", OriginalURL: "https://example.org"},
			},
			userID:  "test_user",
			wantErr: ErrDuplicateCorrID.Error(),
			wantLen: 0,
		},
		{
			name: "empty URL",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: ""},
			},
			userID:  "test_user",
			wantErr: ErrEmptyURL.Error(),
			wantLen: 0,
		},
		{
			name: "successful batch",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "2", OriginalURL: "https://example.org"},
			},
			userID:  "test_user",
			wantErr: "",
			wantLen: 2,
		},
		{
			name: "batch save failure",
			reqs: []models.BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://fail.com"},
				{CorrelationID: "2", OriginalURL: "https://example.org"},
			},
			userID:  "test_user",
			wantErr: "batch save failed",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{store: make(map[string]models.URL)}
			svc := NewService(repo, "http://localhost:8080", "test_secret")

			resp, err := svc.BatchShorten(tt.reqs, tt.userID)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				assert.Nil(t, resp)
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
				assert.Equal(t, tt.reqs[i].OriginalURL, originalURL)
			}
		})
	}
}

func TestJWT(t *testing.T) {
	svc := NewService(nil, "http://localhost:8080", "test_secret")

	// Тест 1: GenerateUserID
	userID, err := svc.GenerateUserID()
	assert.NoError(t, err)
	assert.Len(t, userID, 16, "UserID should be 16 characters long")

	// Тест 2: GenerateJWT и ParseJWT успех
	token, err := svc.GenerateJWT(userID)
	assert.NoError(t, err)
	parsedUserID, err := svc.ParseJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, parsedUserID)

	// Тест 3: ParseJWT с недействительным токеном
	_, err = svc.ParseJWT("invalid_token")
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Тест 4: ParseJWT с истёкшим токеном
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(-time.Hour).Unix(),
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := expiredToken.SignedString([]byte("test_secret"))
	assert.NoError(t, err)
	_, err = svc.ParseJWT(tokenString)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestGetURLsByUserID(t *testing.T) {
	repo := &mockRepository{store: make(map[string]models.URL)}
	svc := NewService(repo, "http://localhost:8080", "test_secret")

	// Добавляем тестовые данные
	repo.store["id1"] = models.URL{ShortID: "id1", OriginalURL: "https://example.com", UserID: "test_user"}
	repo.store["id2"] = models.URL{ShortID: "id2", OriginalURL: "https://test.com", UserID: "test_user"}
	repo.store["id3"] = models.URL{ShortID: "id3", OriginalURL: "https://other.com", UserID: "other_user"}

	// Тест 1: Успех
	urls, err := svc.GetURLsByUserID("test_user")
	assert.NoError(t, err)
	assert.Len(t, urls, 2)
	for _, u := range urls {
		assert.Contains(t, []string{"https://example.com", "https://test.com"}, u.OriginalURL)
		assert.True(t, strings.HasPrefix(u.ShortURL, "http://localhost:8080/"))
	}

	// Тест 2: Пользователь без URL
	urls, err = svc.GetURLsByUserID("unknown_user")
	assert.NoError(t, err)
	assert.Len(t, urls, 0)
}
