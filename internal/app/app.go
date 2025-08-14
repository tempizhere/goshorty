package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
)

// Создаём структуры для JSON
// ShortenRequest представляет запрос на сокращение URL в JSON формате
type ShortenRequest struct {
	URL string `json:"url"` // Оригинальный URL для сокращения
}

// ShortenResponse представляет ответ с сокращённым URL в JSON формате
type ShortenResponse struct {
	Result string `json:"result"` // Сокращённый URL
}

// ExpandResponse представляет ответ с оригинальным URL в JSON формате
type ExpandResponse struct {
	URL string `json:"url"` // Оригинальный URL
}

// App содержит HTTP хендлеры и зависимости для обработки запросов к сервису сокращения URL
type App struct {
	svc    *service.Service    // Сервис для бизнес-логики
	db     repository.Database // Интерфейс для работы с базой данных
	logger *zap.Logger         // Логгер для записи событий
}

// NewApp создаёт новый экземпляр App с указанными зависимостями
func NewApp(svc *service.Service, db repository.Database, logger *zap.Logger) *App {
	return &App{
		svc:    svc,
		db:     db,
		logger: logger,
	}
}

// createShortURL создаёт короткий URL и возвращает его или ошибку
func (a *App) createShortURL(originalURL string, userID string) (string, error) {
	if originalURL == "" {
		return "", errors.New("empty URL")
	}
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return "", errors.New("invalid URL")
	}
	shortURL, err := a.svc.CreateShortURL(originalURL, userID)
	return shortURL, err
}

// HandlePostURL обрабатывает POST-запросы на "/" для сокращения URL через plain text
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

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	originalURL := strings.TrimSpace(string(body))
	shortURL, err := a.createShortURL(originalURL, userID)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			if _, writeErr := w.Write([]byte(shortURL)); writeErr != nil {
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

// HandleGetURL обрабатывает GET-запросы на "/{id}" для получения оригинального URL по короткому ID
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
		u, found := a.svc.Get(id)
		if found && u.DeletedFlag {
			http.Error(w, "URL is deleted", http.StatusGone)
			return
		}
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// HandleJSONShorten обрабатывает POST-запросы на "/api/shorten" для сокращения URL через JSON API
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

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	shortURL, err := a.createShortURL(reqBody.URL, userID)
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

// HandleJSONExpand обрабатывает GET-запросы на "/api/expand/{id}" для получения оригинального URL через JSON API
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

// HandlePing обрабатывает GET-запросы на "/ping" для проверки соединения с базой данных
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

// HandleBatchShorten обрабатывает POST-запросы на "/api/shorten/batch" для пакетного сокращения URL
func (a *App) HandleBatchShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
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

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
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

// HandleUserURLs обрабатывает GET-запросы на "/api/user/urls" для получения всех URL пользователя
func (a *App) HandleUserURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
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

// HandleBatchDeleteURLs обрабатывает DELETE-запросы на "/api/user/urls" для пакетного удаления URL пользователя
func (a *App) HandleBatchDeleteURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Вызываем асинхронное удаление через сервис
	a.svc.BatchDeleteAsync(userID, ids)

	w.WriteHeader(http.StatusAccepted)
}

// Пул буферов для JSON кодирования
var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// writeJSONResponse пишет JSON-ответ с проверкой ошибок
func (a *App) writeJSONResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Используем пул буферов для уменьшения аллокаций
	buf := jsonBufferPool.Get().(*strings.Builder)
	buf.Reset()
	defer jsonBufferPool.Put(buf)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}

	// Убираем перенос строки, который добавляет json.Encoder
	jsonStr := strings.TrimSpace(buf.String())
	if _, err := w.Write([]byte(jsonStr)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
