package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryRepository_GetStats(t *testing.T) {
	repo := NewMemoryRepository()

	// Проверяем, что MemoryRepository реализует интерфейс Repository
	var _ Repository = (*MemoryRepository)(nil)

	t.Run("Empty repository", func(t *testing.T) {
		urls, users, err := repo.GetStats()
		assert.NoError(t, err)
		assert.Equal(t, 0, urls)
		assert.Equal(t, 0, users)
	})

	t.Run("Repository with URLs", func(t *testing.T) {
		// Очищаем репозиторий
		repo.Clear()

		// Добавляем URL для разных пользователей
		_, err := repo.Save("id1", "https://example1.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id2", "https://example2.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id3", "https://example3.com", "user2")
		assert.NoError(t, err)

		urls, users, err := repo.GetStats()
		assert.NoError(t, err)
		assert.Equal(t, 3, urls)
		assert.Equal(t, 2, users)
	})

	t.Run("Repository with deleted URLs", func(t *testing.T) {
		// Очищаем репозиторий
		repo.Clear()

		// Добавляем URL
		_, err := repo.Save("id1", "https://example1.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id2", "https://example2.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id3", "https://example3.com", "user2")
		assert.NoError(t, err)

		// Удаляем один URL
		err = repo.BatchDelete("user1", []string{"id1"})
		assert.NoError(t, err)

		urls, users, err := repo.GetStats()
		assert.NoError(t, err)
		assert.Equal(t, 2, urls)  // Только не удаленные URL
		assert.Equal(t, 2, users) // Пользователи остаются те же
	})

	t.Run("Repository with empty user IDs", func(t *testing.T) {
		// Очищаем репозиторий
		repo.Clear()

		// Добавляем URL с пустым userID
		_, err := repo.Save("id1", "https://example1.com", "")
		assert.NoError(t, err)
		_, err = repo.Save("id2", "https://example2.com", "user1")
		assert.NoError(t, err)

		urls, users, err := repo.GetStats()
		assert.NoError(t, err)
		assert.Equal(t, 2, urls)
		assert.Equal(t, 1, users) // Только user1, пустой userID не считается
	})

	t.Run("Repository with all deleted URLs", func(t *testing.T) {
		// Очищаем репозиторий
		repo.Clear()

		// Добавляем URL
		_, err := repo.Save("id1", "https://example1.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id2", "https://example2.com", "user1")
		assert.NoError(t, err)

		// Удаляем все URL
		err = repo.BatchDelete("user1", []string{"id1", "id2"})
		assert.NoError(t, err)

		urls, users, err := repo.GetStats()
		assert.NoError(t, err)
		assert.Equal(t, 0, urls)
		assert.Equal(t, 0, users) // Нет активных URL, значит нет пользователей
	})
}
