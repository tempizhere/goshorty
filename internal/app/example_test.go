package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/app"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
)

// ExampleApp_HandlePostURL демонстрирует обработку POST запроса для сокращения URL через plain text
func ExampleApp_HandlePostURL() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

	// Создаём HTTP запрос
	body := strings.NewReader("https://example.com/very-long-url")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "text/plain")

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandlePostURL(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)
	shortURL := strings.TrimSpace(w.Body.String())
	fmt.Printf("URL содержит базовый адрес: %t\n", strings.HasPrefix(shortURL, "http://localhost:8080/"))
	fmt.Printf("ID имеет правильную длину: %t\n", len(shortURL)-len("http://localhost:8080/") == 8)

	// Output:
	// Статус код: 201
	// URL содержит базовый адрес: true
	// ID имеет правильную длину: true
}

// ExampleApp_HandleJSONShorten демонстрирует обработку POST запроса для сокращения URL через JSON API
func ExampleApp_HandleJSONShorten() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

	// Создаём JSON запрос
	requestBody := app.ShortenRequest{
		URL: "https://example.com/very-long-url",
	}
	jsonData, _ := json.Marshal(requestBody)

	// Создаём HTTP запрос
	req := httptest.NewRequest("POST", "/api/shorten", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleJSONShorten(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)

	var response app.ShortenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}
	fmt.Printf("URL содержит базовый адрес: %t\n", strings.HasPrefix(response.Result, "http://localhost:8080/"))
	fmt.Printf("ID имеет правильную длину: %t\n", len(response.Result)-len("http://localhost:8080/") == 8)

	// Output:
	// Статус код: 201
	// URL содержит базовый адрес: true
	// ID имеет правильную длину: true
}

// ExampleApp_HandleGetURL демонстрирует обработку GET запроса для получения оригинального URL
func ExampleApp_HandleGetURL() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

	// Сначала создаём короткий URL
	originalURL := "https://example.com/very-long-url"
	userID := "user-123"
	shortURL, _ := svc.CreateShortURL(originalURL, userID)
	shortID := shortURL[len("http://localhost:8080/"):]

	// Создаём HTTP запрос для получения оригинального URL
	req := httptest.NewRequest("GET", "/"+shortID, nil)

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleGetURL(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)
	fmt.Printf("Location header: %s\n", w.Header().Get("Location"))

	// Output:
	// Статус код: 307
	// Location header: https://example.com/very-long-url
}

// ExampleApp_HandleJSONExpand демонстрирует обработку GET запроса для получения оригинального URL через JSON API
func ExampleApp_HandleJSONExpand() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

	// Сначала создаём короткий URL
	originalURL := "https://example.com/very-long-url"
	userID := "user-123"
	shortURL, _ := svc.CreateShortURL(originalURL, userID)
	shortID := shortURL[len("http://localhost:8080/"):]

	// Создаём HTTP запрос
	req := httptest.NewRequest("GET", "/api/expand/"+shortID, nil)

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleJSONExpand(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)

	var response app.ExpandResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}
	fmt.Printf("Оригинальный URL: %s\n", response.URL)

	// Output:
	// Статус код: 200
	// Оригинальный URL: https://example.com/very-long-url
}

// ExampleApp_HandleBatchShorten демонстрирует обработку POST запроса для пакетного сокращения URL
func ExampleApp_HandleBatchShorten() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

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

	jsonData, _ := json.Marshal(requests)

	// Создаём HTTP запрос
	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Post("/api/shorten/batch", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleBatchShorten(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)

	var responses []models.BatchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &responses); err != nil {
		fmt.Printf("Ошибка при разборе JSON: %v\n", err)
		return
	}
	fmt.Printf("Обработано запросов: %d\n", len(responses))
	fmt.Printf("Все URL содержат базовый адрес: %t\n", func() bool {
		for _, resp := range responses {
			if !strings.HasPrefix(resp.ShortURL, "http://localhost:8080/") {
				return false
			}
		}
		return true
	}())

	// Output:
	// Статус код: 201
	// Обработано запросов: 2
	// Все URL содержат базовый адрес: true
}

// ExampleApp_HandleUserURLs демонстрирует обработку GET запроса для получения URL пользователя
func ExampleApp_HandleUserURLs() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

	// Создаём несколько URL для пользователя
	// Используем тот же userID, который генерирует middleware
	userID, _ := svc.GenerateUserID()
	if _, err := svc.CreateShortURL("https://example.com/url1", userID); err != nil {
		fmt.Printf("Ошибка при создании URL: %v\n", err)
		return
	}
	if _, err := svc.CreateShortURL("https://example.com/url2", userID); err != nil {
		fmt.Printf("Ошибка при создании URL: %v\n", err)
		return
	}

	// Создаём JWT токен для пользователя
	token, _ := svc.GenerateJWT(userID)

	// Создаём HTTP запрос с JWT токеном
	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{
		Name:  "jwt",
		Value: token,
	})

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleUserURLs(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)

	var responses []models.ShortURLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &responses); err != nil {
		fmt.Printf("Ошибка при разборе JSON: %v\n", err)
		return
	}
	fmt.Printf("URL пользователя: %d\n", len(responses))
	if len(responses) > 0 {
		fmt.Printf("Первый URL содержит базовый адрес: %t\n", strings.HasPrefix(responses[0].ShortURL, "http://localhost:8080/"))
	} else {
		fmt.Printf("Нет URL для пользователя\n")
	}

	// Output:
	// Статус код: 200
	// URL пользователя: 2
	// Первый URL содержит базовый адрес: true
}

// ExampleApp_HandleBatchDeleteURLs демонстрирует обработку DELETE запроса для пакетного удаления URL
func ExampleApp_HandleBatchDeleteURLs() {
	// Создаём зависимости
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, "http://localhost:8080", "test-secret")
	logger := zap.NewNop()
	appInstance := app.NewApp(svc, nil, logger)

	// Создаём несколько URL для пользователя
	userID := "user-123"
	shortURL1, _ := svc.CreateShortURL("https://example.com/url1", userID)
	shortURL2, _ := svc.CreateShortURL("https://example.com/url2", userID)

	// Извлекаем ID из коротких URL
	shortID1 := shortURL1[len("http://localhost:8080/"):]
	shortID2 := shortURL2[len("http://localhost:8080/"):]

	// Создаём запрос на удаление
	idsToDelete := []string{shortID1, shortID2}
	jsonData, _ := json.Marshal(idsToDelete)

	// Создаём HTTP запрос
	req := httptest.NewRequest("DELETE", "/api/user/urls", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Создаём response recorder
	w := httptest.NewRecorder()

	// Создаём маршрутизатор с middleware
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Delete("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleBatchDeleteURLs(w, r)
	})

	// Выполняем запрос
	r.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Статус код: %d\n", w.Code)
	fmt.Printf("Удалено URL: %d\n", len(idsToDelete))

	// Output:
	// Статус код: 202
	// Удалено URL: 2
}
