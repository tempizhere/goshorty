package service_test

import (
	"fmt"

	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
)

// ExampleService_GenerateShortID демонстрирует генерацию короткого ID
func ExampleService_GenerateShortID() {
	// Создаём сервис с in-memory репозиторием
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")

	// Генерируем короткий ID
	id, err := svc.GenerateShortID()
	if err != nil {
		fmt.Printf("Ошибка генерации ID: %v\n", err)
		return
	}

	fmt.Printf("Длина ID: %d символов\n", len(id))
	fmt.Printf("ID содержит только допустимые символы: %t\n", func() bool {
		for _, char := range id {
			if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
				return false
			}
		}
		return true
	}())

	// Output:
	// Длина ID: 8 символов
	// ID содержит только допустимые символы: true
}

// ExampleService_CreateShortURL демонстрирует создание короткого URL
func ExampleService_CreateShortURL() {
	// Создаём сервис с in-memory репозиторием
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")

	// Создаём короткий URL
	originalURL := "https://example.com/very-long-url"
	userID := "user-123"

	shortURL, err := svc.CreateShortURL(originalURL, userID)
	if err != nil {
		fmt.Printf("Ошибка создания URL: %v\n", err)
		return
	}

	fmt.Printf("Оригинальный URL: %s\n", originalURL)
	fmt.Printf("URL содержит базовый адрес: %t\n", len(shortURL) > len("http://localhost:8080/"))
	fmt.Printf("ID имеет правильную длину: %t\n", len(shortURL)-len("http://localhost:8080/") == 8)

	// Output:
	// Оригинальный URL: https://example.com/very-long-url
	// URL содержит базовый адрес: true
	// ID имеет правильную длину: true
}

// ExampleService_GetOriginalURL демонстрирует получение оригинального URL
func ExampleService_GetOriginalURL() {
	// Создаём сервис с in-memory репозиторием
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")

	// Создаём короткий URL
	originalURL := "https://example.com/very-long-url"
	userID := "user-123"

	shortURL, _ := svc.CreateShortURL(originalURL, userID)

	// Извлекаем ID из короткого URL
	shortID := shortURL[len("http://localhost:8080/"):]

	// Получаем оригинальный URL
	retrievedURL, exists := svc.GetOriginalURL(shortID)
	if !exists {
		fmt.Println("URL не найден")
		return
	}

	fmt.Printf("Оригинальный URL: %s\n", retrievedURL)
	fmt.Printf("URL совпадает: %t\n", retrievedURL == originalURL)

	// Output:
	// Оригинальный URL: https://example.com/very-long-url
	// URL совпадает: true
}

// ExampleService_BatchShorten демонстрирует пакетное сокращение URL
func ExampleService_BatchShorten() {
	// Создаём сервис с in-memory репозиторием
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")

	// Создаём пакет запросов
	requests := []models.BatchRequest{
		{
			CorrelationID: "req-1",
			OriginalURL:   "https://example.com/url1",
		},
		{
			CorrelationID: "req-2",
			OriginalURL:   "https://example.com/url2",
		},
	}

	userID := "user-123"

	// Выполняем пакетное сокращение
	responses, err := svc.BatchShorten(requests, userID)
	if err != nil {
		fmt.Printf("Ошибка пакетного сокращения: %v\n", err)
		return
	}

	fmt.Printf("Обработано запросов: %d\n", len(responses))
	fmt.Printf("Все URL содержат базовый адрес: %t\n", func() bool {
		for _, resp := range responses {
			if len(resp.ShortURL) <= len("http://localhost:8080/") {
				return false
			}
		}
		return true
	}())

	// Output:
	// Обработано запросов: 2
	// Все URL содержат базовый адрес: true
}

// ExampleService_GenerateJWT демонстрирует генерацию JWT токена
func ExampleService_GenerateJWT() {
	// Создаём сервис
	svc := service.NewService(nil, "http://localhost:8080", "test-secret")

	// Генерируем JWT токен
	userID := "user-123"
	token, err := svc.GenerateJWT(userID)
	if err != nil {
		fmt.Printf("Ошибка генерации JWT: %v\n", err)
		return
	}

	fmt.Printf("UserID: %s\n", userID)
	fmt.Printf("JWT токен сгенерирован: %t\n", len(token) > 0)
	fmt.Printf("Длина токена: %d символов\n", len(token))

	// Output:
	// UserID: user-123
	// JWT токен сгенерирован: true
	// Длина токена: 133 символов
}

// ExampleService_ParseJWT демонстрирует парсинг JWT токена
func ExampleService_ParseJWT() {
	// Создаём сервис
	svc := service.NewService(nil, "http://localhost:8080", "test-secret")

	// Генерируем JWT токен
	userID := "user-123"
	token, _ := svc.GenerateJWT(userID)

	// Парсим JWT токен
	parsedUserID, err := svc.ParseJWT(token)
	if err != nil {
		fmt.Printf("Ошибка парсинга JWT: %v\n", err)
		return
	}

	fmt.Printf("Исходный UserID: %s\n", userID)
	fmt.Printf("Извлечённый UserID: %s\n", parsedUserID)
	fmt.Printf("UserID совпадает: %t\n", parsedUserID == userID)

	// Output:
	// Исходный UserID: user-123
	// Извлечённый UserID: user-123
	// UserID совпадает: true
}
