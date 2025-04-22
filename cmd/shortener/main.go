package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

// Хранилище для пар "короткий ID — URL"
var urlStore = make(map[string]string)

func main() {
	// Создаём маршрутизатор
	mux := http.NewServeMux()

	// Регистрируем обработчики
	mux.HandleFunc("/", handlePostURL)
	mux.HandleFunc("/{id}", handleGetURL)

	// Запускаем сервер на порту 8080
	err := http.ListenAndServe(":8080", mux)
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
	if r.Header.Get("Content-Type") != "text/plain" {
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
	id := generateShortID()
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
	id := r.URL.Path[1:]
	originalURL, exists := urlStore[id]
	if !exists {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Генерирует короткий ID из URL
func generateShortID() string {
	// Генерируем случайные 8 байт
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	// Кодируем в base64 и обрезаем
	encoded := base64.URLEncoding.EncodeToString(bytes)
	return encoded[:8]
}
