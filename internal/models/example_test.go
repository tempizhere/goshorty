package models_test

import (
	"encoding/json"
	"fmt"

	"github.com/tempizhere/goshorty/internal/models"
)

// ExampleBatchRequest демонстрирует создание запроса на пакетное сокращение URL
func ExampleBatchRequest() {
	// Создаём запрос на сокращение URL
	req := models.BatchRequest{
		CorrelationID: "req-1",
		OriginalURL:   "https://example.com/very-long-url",
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(req)
	fmt.Printf("JSON запрос: %s\n", jsonData)

	// Output:
	// JSON запрос: {"correlation_id":"req-1","original_url":"https://example.com/very-long-url"}
}

// ExampleBatchResponse демонстрирует создание ответа на пакетное сокращение URL
func ExampleBatchResponse() {
	// Создаём ответ с сокращённым URL
	resp := models.BatchResponse{
		CorrelationID: "req-1",
		ShortURL:      "http://localhost:8080/abc123",
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(resp)
	fmt.Printf("JSON ответ: %s\n", jsonData)

	// Output:
	// JSON ответ: {"correlation_id":"req-1","short_url":"http://localhost:8080/abc123"}
}

// ExampleURL демонстрирует создание структуры URL
func ExampleURL() {
	// Создаём URL с полной информацией
	url := models.URL{
		ShortID:     "abc123",
		OriginalURL: "https://example.com/very-long-url",
		UserID:      "user-456",
		DeletedFlag: false,
	}

	fmt.Printf("Короткий ID: %s\n", url.ShortID)
	fmt.Printf("Оригинальный URL: %s\n", url.OriginalURL)
	fmt.Printf("Пользователь: %s\n", url.UserID)
	fmt.Printf("Удалён: %t\n", url.DeletedFlag)

	// Output:
	// Короткий ID: abc123
	// Оригинальный URL: https://example.com/very-long-url
	// Пользователь: user-456
	// Удалён: false
}

// ExampleShortURLResponse демонстрирует создание ответа с информацией о сокращённом URL
func ExampleShortURLResponse() {
	// Создаём ответ для API
	resp := models.ShortURLResponse{
		ShortURL:    "http://localhost:8080/abc123",
		OriginalURL: "https://example.com/very-long-url",
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(resp)
	fmt.Printf("JSON ответ: %s\n", jsonData)

	// Output:
	// JSON ответ: {"short_url":"http://localhost:8080/abc123","original_url":"https://example.com/very-long-url"}
}
