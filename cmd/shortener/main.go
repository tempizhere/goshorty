package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/cmd/shortener/config"
	"strings"
	"encoding/json"
)

// Хранилище для пар "короткий ID — URL"
var urlStore = make(map[string]string)

// Структуры для JSON-запросов и ответов
type ShortenRequest struct {
    URL string `json:"url"`
}
type ShortenResponse struct {
    Result string `json:"result"`
}

func main() {
	// Получаем конфигурацию
	cfg := config.NewConfig()

	// Создаём маршрутизатор
	r := chi.NewRouter()

	// Регистрируем обработчики
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlePostURL(w, r, cfg)
	})
	r.Get("/{id}", handleGetURL)

    // Маршрут для обработки JSON-запросов на создание короткого URL
    r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
        handleJSONShorten(w, r, cfg)
    })

    // Маршрут для обработки запросов на расширение URL
    r.Get("/api/expand", handleJSONExpand)

	// Запускаем сервер на порту 8080
	err := http.ListenAndServe(cfg.RunAddr, r)
	if err != nil {
		panic(err)
	}
}

// Обработчик POST-запросов на "/", принимает URL в теле запроса и возвращает короткий URL
func handlePostURL(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "text/plain") {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	originalURL := string(body)
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}
	id, err := generateShortID()
	if err != nil {
		http.Error(w, "Failed to generate ID", http.StatusInternalServerError)
		return
	}
	urlStore[id] = originalURL
	shortURL := fmt.Sprintf("%s/%s", cfg.BaseURL, id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, shortURL)
}

// Обработчик GET-запросов на "/api/expand", возвращает редирект на исходный URL
func handleJSONExpand(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "ID parameter is required", http.StatusBadRequest)
        return
    }

    originalURL, exists := urlStore[id]
    if !exists {
        http.Error(w, "URL not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Location", originalURL)
    w.WriteHeader(http.StatusTemporaryRedirect)
}

// Обработчик GET-запросов на "/{id}"
func handleGetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	originalURL, exists := urlStore[id]
	if !exists {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Обработчик POST-запросов на "/api/shorten", принимает JSON-запрос и возвращает JSON-ответ
func handleJSONShorten(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    contentType := r.Header.Get("Content-Type")
    if !strings.HasPrefix(contentType, "application/json") {
        http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
        return
    }

    var req ShortenRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
        return
    }
    if req.URL == "" {
        http.Error(w, "URL is required", http.StatusBadRequest)
        return
    }

    id := generateShortID(req.URL)
    urlStore[id] = req.URL
    shortURL := fmt.Sprintf("%s/%s", cfg.BaseURL, id)

    resp := ShortenResponse{Result: shortURL}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
}

// Генерирует короткий ID из URL
func generateShortID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(bytes)
	return encoded[:8], nil
}
