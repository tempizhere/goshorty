package app

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/service"
	"io"
	"net/http"
	"strings"
)

// Создаём структуры для JSON
type ShortenRequest struct {
	URL string `json:"url"`
}
type ShortenResponse struct {
	Result string `json:"result"`
}
type ExpandResponse struct {
	URL string `json:"url"`
}

// App содержит хендлеры и зависимости
type App struct {
	svc *service.Service
}

// NewApp создаёт новый экземпляр App
func NewApp(svc *service.Service) *App {
	return &App{svc: svc}
}

// createShortURL создаёт короткий URL и возвращает его или ошибку
func (a *App) createShortURL(originalURL string) (string, error) {
	if originalURL == "" {
		return "", errors.New("empty URL")
	}
	shortURL, err := a.svc.CreateShortURL(originalURL)
	if err != nil {
		return "", err
	}
	return shortURL, nil
}

// Обработчик POST-запросов на "/"
func (a *App) HandlePostURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	// Проверяем Content-Type для сжатых запросов
	if r.Header.Get("Content-Encoding") == "gzip" &&
		!strings.Contains(r.Header.Get("Content-Type"), "text/plain") &&
		!strings.Contains(r.Header.Get("Content-Type"), "application/x-gzip") {
		http.Error(w, "Invalid Content-Type for gzip request", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	shortURL, err := a.createShortURL(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

// Обработчик GET-запросов на "/{id}"
func (a *App) HandleGetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing URL ID", http.StatusBadRequest)
		return
	}
	originalURL, exists := a.svc.GetOriginalURL(id)
	if !exists {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Обработчик POST-запросов на "/api/shorten"
func (a *App) HandleJSONShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}
	// Проверяем, что запрос не сжат некорректно
	if r.Header.Get("Content-Encoding") == "gzip" && !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Invalid Content-Type for gzip request", http.StatusBadRequest)
		return
	}
	var reqBody ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	shortURL, err := a.createShortURL(reqBody.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respBody := ShortenResponse{
		Result: shortURL,
	}
	a.writeJSONResponse(w, http.StatusCreated, respBody)
}

// Обработчик GET-запросов на "/api/expand/{id}"
func (a *App) HandleJSONExpand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	originalURL, exists := a.svc.GetOriginalURL(id)
	if !exists {
		a.writeJSONResponse(w, http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: "URL not found"})
		return
	}
	respBody := ExpandResponse{
		URL: originalURL,
	}
	a.writeJSONResponse(w, http.StatusOK, respBody)
}

// writeJSONResponse пишет JSON-ответ с проверкой ошибок
func (a *App) writeJSONResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
