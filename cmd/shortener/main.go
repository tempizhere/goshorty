package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"github.com/go-chi/chi/v5"
)

// Хранилище для пар "короткий ID — URL"
var urlStore = make(map[string]string)

func main() {
	// Создаём маршрутизатор
	r := chi.NewRouter()

	// Регистрируем обработчики
	r.Post("/", handlePostURL)
	r.Get("/{id}", handleGetURL)

	// Запускаем сервер на порту 8080
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		panic(err)
	}
}

// Обработчик POST-запросов на "/"
func handlePostURL(w http.ResponseWriter, r *http.Request) {
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
	shortURL := fmt.Sprintf("http://localhost:8080/%s", id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, shortURL)
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
