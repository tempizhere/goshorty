package repository_test

import (
	"fmt"

	"github.com/tempizhere/goshorty/internal/repository"
)

// ExampleMemoryRepository_Save демонстрирует сохранение URL в in-memory репозитории
func ExampleMemoryRepository_Save() {
	// Создаём in-memory репозиторий
	repo := repository.NewMemoryRepository()

	// Сохраняем URL
	shortID, err := repo.Save("abc123", "https://example.com/very-long-url", "user-123")
	if err != nil {
		fmt.Printf("Ошибка сохранения: %v\n", err)
		return
	}

	fmt.Printf("Сохранён URL с ID: %s\n", shortID)

	// Output:
	// Сохранён URL с ID: abc123
}

// ExampleMemoryRepository_Get демонстрирует получение URL из in-memory репозитория
func ExampleMemoryRepository_Get() {
	// Создаём in-memory репозиторий
	repo := repository.NewMemoryRepository()

	// Сохраняем URL
	repo.Save("abc123", "https://example.com/very-long-url", "user-123")

	// Получаем URL
	url, exists := repo.Get("abc123")
	if !exists {
		fmt.Println("URL не найден")
		return
	}

	fmt.Printf("Короткий ID: %s\n", url.ShortID)
	fmt.Printf("Оригинальный URL: %s\n", url.OriginalURL)
	fmt.Printf("Пользователь: %s\n", url.UserID)

	// Output:
	// Короткий ID: abc123
	// Оригинальный URL: https://example.com/very-long-url
	// Пользователь: user-123
}

// ExampleMemoryRepository_BatchSave демонстрирует пакетное сохранение URL
func ExampleMemoryRepository_BatchSave() {
	// Создаём in-memory репозиторий
	repo := repository.NewMemoryRepository()

	// Создаём пакет URL для сохранения
	urls := map[string]string{
		"abc123": "https://example.com/url1",
		"def456": "https://example.com/url2",
		"ghi789": "https://example.com/url3",
	}

	userID := "user-123"

	// Сохраняем пакет URL
	err := repo.BatchSave(urls, userID)
	if err != nil {
		fmt.Printf("Ошибка пакетного сохранения: %v\n", err)
		return
	}

	fmt.Printf("Сохранено URL: %d\n", len(urls))

	// Проверяем сохранение
	count := 0
	for shortID := range urls {
		_, exists := repo.Get(shortID)
		if exists {
			count++
		}
	}
	fmt.Printf("Успешно сохранено URL: %d\n", count)

	// Output:
	// Сохранено URL: 3
	// Успешно сохранено URL: 3
}

// ExampleMemoryRepository_GetURLsByUserID демонстрирует получение URL пользователя
func ExampleMemoryRepository_GetURLsByUserID() {
	// Создаём in-memory репозиторий
	repo := repository.NewMemoryRepository()

	// Сохраняем URL для разных пользователей
	repo.Save("abc123", "https://example.com/url1", "user-123")
	repo.Save("def456", "https://example.com/url2", "user-123")
	repo.Save("ghi789", "https://example.com/url3", "user-456")

	// Получаем URL пользователя user-123
	urls, err := repo.GetURLsByUserID("user-123")
	if err != nil {
		fmt.Printf("Ошибка получения URL: %v\n", err)
		return
	}

	fmt.Printf("URL пользователя user-123: %d\n", len(urls))
	fmt.Printf("Все URL имеют правильный формат: %t\n", func() bool {
		for _, url := range urls {
			if url.ShortID == "" || url.OriginalURL == "" {
				return false
			}
		}
		return true
	}())

	// Output:
	// URL пользователя user-123: 2
	// Все URL имеют правильный формат: true
}

// ExampleMemoryRepository_BatchDelete демонстрирует пакетное удаление URL
func ExampleMemoryRepository_BatchDelete() {
	// Создаём in-memory репозиторий
	repo := repository.NewMemoryRepository()

	// Сохраняем URL
	repo.Save("abc123", "https://example.com/url1", "user-123")
	repo.Save("def456", "https://example.com/url2", "user-123")
	repo.Save("ghi789", "https://example.com/url3", "user-123")

	// Удаляем URL
	idsToDelete := []string{"abc123", "def456"}
	err := repo.BatchDelete("user-123", idsToDelete)
	if err != nil {
		fmt.Printf("Ошибка удаления: %v\n", err)
		return
	}

	fmt.Printf("Удалено URL: %d\n", len(idsToDelete))

	// Проверяем статус URL
	for _, id := range idsToDelete {
		url, exists := repo.Get(id)
		if exists {
			fmt.Printf("URL %s удалён: %t\n", id, url.DeletedFlag)
		}
	}

	// Output:
	// Удалено URL: 2
	// URL abc123 удалён: true
	// URL def456 удалён: true
}

// ExampleMemoryRepository_Clear демонстрирует очистку репозитория
func ExampleMemoryRepository_Clear() {
	// Создаём in-memory репозиторий
	repo := repository.NewMemoryRepository()

	// Сохраняем URL
	repo.Save("abc123", "https://example.com/url1", "user-123")
	repo.Save("def456", "https://example.com/url2", "user-123")

	// Проверяем количество URL
	_, exists1 := repo.Get("abc123")
	_, exists2 := repo.Get("def456")
	fmt.Printf("До очистки: abc123=%t, def456=%t\n", exists1, exists2)

	// Очищаем репозиторий
	repo.Clear()

	// Проверяем после очистки
	_, exists1 = repo.Get("abc123")
	_, exists2 = repo.Get("def456")
	fmt.Printf("После очистки: abc123=%t, def456=%t\n", exists1, exists2)

	// Output:
	// До очистки: abc123=true, def456=true
	// После очистки: abc123=false, def456=false
}
