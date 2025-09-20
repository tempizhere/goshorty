package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/models"
)

func TestService_GetStats(t *testing.T) {
	t.Run("GetStats with mock repository", func(t *testing.T) {
		// Создаем мок-репозиторий с тестовыми данными
		mockRepo := &mockRepository{store: make(map[string]models.URL)}

		// Добавляем тестовые данные
		_, err := mockRepo.Save("id1", "https://example1.com", "user1")
		assert.NoError(t, err)
		_, err = mockRepo.Save("id2", "https://example2.com", "user1")
		assert.NoError(t, err)
		_, err = mockRepo.Save("id3", "https://example3.com", "user2")
		assert.NoError(t, err)

		// Создаем сервис
		svc := NewService(mockRepo, "http://localhost:8080", "test_secret")

		// Вызываем метод
		urls, users, err := svc.GetStats()

		// Проверяем результат
		assert.NoError(t, err)
		assert.Equal(t, 3, urls)
		assert.Equal(t, 2, users)
	})

	t.Run("GetStats with empty repository", func(t *testing.T) {
		// Создаем пустой мок-репозиторий
		mockRepo := &mockRepository{store: make(map[string]models.URL)}

		// Создаем сервис
		svc := NewService(mockRepo, "http://localhost:8080", "test_secret")

		// Вызываем метод
		urls, users, err := svc.GetStats()

		// Проверяем результат
		assert.NoError(t, err)
		assert.Equal(t, 0, urls)
		assert.Equal(t, 0, users)
	})
}
