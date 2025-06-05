package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/config"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
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
	db  repository.Database
	cfg *config.Config
}

// NewApp создаёт новый экземпляр App
func NewApp(svc *service.Service, db repository.Database, cfg *config.Config) *App {
	return &App{svc: svc, db: db, cfg: cfg}
}

// createShortURL создаёт короткий URL и возвращает его или ошибку
func (a *App) createShortURL(r *http.Request, originalURL string) (string, error) {
	if originalURL == "" {
		return "", errors.New("empty URL")
	}
	userID, ok := middleware.GetUserID(r)
	if !ok || userID == "" {
		return "", errors.New("no user ID")
	}
	shortURL, err := a.svc.CreateShortURL(originalURL, userID)
	if err != nil {
		return shortURL, err
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
	shortURL, err := a.createShortURL(r, string(body))
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			if _, err := w.Write([]byte(shortURL)); err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
			}
			return
		}
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
	shortURL, err := a.createShortURL(r, reqBody.URL)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			respBody := ShortenResponse{
				Result: shortURL,
			}
			a.writeJSONResponse(w, http.StatusConflict, respBody)
			return
		}
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

// Обработчик GET-запросов на "/ping"
func (a *App) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if a.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}
	if err := a.db.Ping(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Обработчик POST-запросов на "/api/shorten/batch"
func (a *App) HandleBatchShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}
	userID, ok := middleware.GetUserID(r)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var reqBody []models.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(reqBody) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}
	for _, req := range reqBody {
		if req.CorrelationID == "" {
			http.Error(w, "Missing correlation_id", http.StatusBadRequest)
			return
		}
		if _, err := url.ParseRequestURI(req.OriginalURL); err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
	}
	respBody, err := a.svc.BatchShorten(reqBody, userID)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			a.writeJSONResponse(w, http.StatusConflict, respBody)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.writeJSONResponse(w, http.StatusCreated, respBody)
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

// HandleUserURLs возвращает все URL пользователя
func (a *App) HandleUserURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := a.svc.GetURLsByUserID(userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.writeJSONResponse(w, http.StatusOK, urls)
}
