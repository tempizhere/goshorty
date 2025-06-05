package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/config"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
)

// errorReader симулирует ошибку чтения
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// compressData сжимает данные с помощью Gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Тесты для HandlePostURL и HandleJSONShorten
func TestHandlePostURL(t *testing.T) {
	// Создаём временный файл для тестов
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Создаём зависимости
	cfg := &config.Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test_secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	appInstance := NewApp(svc, nil, cfg)

	// Таблица тестов
	tests := []struct {
		name            string
		method          string
		url             string
		contentType     string
		body            io.Reader
		isJSON          bool
		useGzipRequest  bool
		useGzipResponse bool
		largeResponse   bool
		storeSetup      func()
		userID          string
		expectedCode    int
		expectedBody    string
		expectedStored  bool
		expectGzip      bool
	}{
		{
			name:           "Success",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "text/plain",
			body:           strings.NewReader("https://example.com"),
			isJSON:         false,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedStored: true,
			expectGzip:     false,
		},
		{
			name:        "Duplicate URL",
			method:      http.MethodPost,
			url:         "/",
			contentType: "text/plain",
			body:        strings.NewReader("https://example.com"),
			isJSON:      false,
			userID:      "test_user",
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "test_user")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode:   http.StatusConflict,
			expectedStored: true,
			expectedBody:   "http://localhost:8080/testID",
			expectGzip:     false,
		},
		{
			name:           "InvalidMethod",
			method:         http.MethodGet,
			url:            "/",
			contentType:    "text/plain",
			body:           nil,
			isJSON:         false,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Method not allowed\n",
			expectedStored: false,
			expectGzip:     false,
		},
		{
			name:           "InvalidContentType",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "application/json",
			body:           strings.NewReader("https://example.com"),
			isJSON:         false,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedStored: true,
			expectGzip:     false,
		},
		{
			name:           "EmptyBody",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "text/plain",
			body:           strings.NewReader(""),
			isJSON:         false,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "empty URL\n",
			expectedStored: false,
			expectGzip:     false,
		},
		{
			name:           "ReadBodyError",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "text/plain",
			body:           strings.NewReader("https://example.com"),
			isJSON:         false,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Failed to read request body\n",
			expectedStored: false,
			expectGzip:     false,
		},
		{
			name:           "JSONSuccess",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:         true,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedBody:   `{"result":"` + cfg.BaseURL + "/",
			expectedStored: true,
			expectGzip:     false,
		},
		{
			name:        "JSONDuplicateURL",
			method:      http.MethodPost,
			url:         "/api/shorten",
			contentType: "application/json",
			body:        strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:      true,
			userID:      "test_user",
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "test_user")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode:   http.StatusConflict,
			expectedBody:   `{"result":"http://localhost:8080/testID"}`,
			expectedStored: true,
			expectGzip:     false,
		},
		{
			name:           "JSONInvalid",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           strings.NewReader(`{invalid json}`),
			isJSON:         true,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
			expectedStored: false,
			expectGzip:     false,
		},
		{
			name:           "JSONEmptyURL",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           strings.NewReader(`{"url":""}`),
			isJSON:         true,
			userID:         "test_user",
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "empty URL\n",
			expectedStored: false,
			expectGzip:     false,
		},
		{
			name:           "GzipRequestJSONSuccess",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           nil,
			isJSON:         true,
			userID:         "test_user",
			useGzipRequest: true,
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedBody:   `{"result":"` + cfg.BaseURL + "/",
			expectedStored: true,
			expectGzip:     false,
		},
		{
			name:           "GzipRequestTextSuccess",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "application/x-gzip",
			body:           nil,
			isJSON:         false,
			userID:         "test_user",
			useGzipRequest: true,
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedStored: true,
			expectGzip:     false,
		},
		{
			name:            "GzipResponseJSONSuccessLarge",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:          true,
			userID:          "test_user",
			useGzipResponse: true,
			largeResponse:   true,
			storeSetup:      func() {},
			expectedCode:    http.StatusCreated,
			expectedBody:    `{"result":"` + cfg.BaseURL + "/",
			expectedStored:  true,
			expectGzip:      true,
		},
		{
			name:            "GzipResponseJSONSmall",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:          true,
			userID:          "test_user",
			useGzipResponse: true,
			largeResponse:   false,
			storeSetup:      func() {},
			expectedCode:    http.StatusCreated,
			expectedBody:    `{"result":"` + cfg.BaseURL + "/",
			expectedStored:  true,
			expectGzip:      false,
		},
		{
			name:            "GzipResponseTextPlain",
			method:          http.MethodPost,
			url:             "/",
			contentType:     "text/plain",
			body:            strings.NewReader("https://example.com"),
			isJSON:          false,
			userID:          "test_user",
			useGzipResponse: true,
			largeResponse:   true,
			storeSetup:      func() {},
			expectedCode:    http.StatusCreated,
			expectedStored:  true,
			expectGzip:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			tt.storeSetup()

			// Подготавливаем сжатое тело для GzipRequest
			var requestBody = tt.body
			if tt.useGzipRequest {
				data := `{"url":"https://example.com"}`
				if !tt.isJSON {
					data = "https://example.com"
				}
				compressed, err := compressData([]byte(data))
				assert.NoError(t, err, "Failed to compress request body")
				requestBody = bytes.NewReader(compressed)
			}

			// Создаём запрос
			req := httptest.NewRequest(tt.method, tt.url, requestBody)
			req.Header.Set("Content-Type", tt.contentType)
			if tt.useGzipRequest {
				req.Header.Set("Content-Encoding", "gzip")
			}
			if tt.useGzipResponse {
				req.Header.Set("Accept-Encoding", "gzip")
			}
			// Добавляем UserID в контекст
			ctx := context.Background()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, middleware.UserIDKey{}, tt.userID)
			}
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			// Для ReadBodyError подменяем тело запроса
			if tt.name == "ReadBodyError" {
				req.Body = io.NopCloser(&errorReader{})
			}

			// Создаём маршрутизатор с GzipMiddleware
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
			r.Use(middleware.AuthMiddleware(svc, cfg, zap.NewNop()))
			if tt.isJSON {
				r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
					if tt.largeResponse {
						// Создаём большой ответ (>1400 байт)
						if r.Method != http.MethodPost {
							http.Error(w, "Method not allowed", http.StatusBadRequest)
							return
						}
						if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
							http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
							return
						}
						var reqBody ShortenRequest
						if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
							http.Error(w, "Invalid JSON", http.StatusBadRequest)
							return
						}
						shortURL, err := appInstance.createShortURL(r, reqBody.URL)
						if err != nil {
							if errors.Is(err, repository.ErrURLExists) {
								respBody := ShortenResponse{
									Result: shortURL,
								}
								w.Header().Set("Content-Type", "application/json")
								w.WriteHeader(http.StatusConflict)
								data, err := json.Marshal(respBody)
								if err != nil {
									http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
									return
								}
								if _, err := w.Write(data); err != nil {
									http.Error(w, "Failed to write response", http.StatusInternalServerError)
								}
								return
							}
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						respBody := struct {
							Result string `json:"result"`
							Filler string `json:"filler"`
						}{
							Result: shortURL,
							Filler: strings.Repeat("x", 1400),
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusCreated)
						data, err := json.Marshal(respBody)
						if err != nil {
							http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
							return
						}
						if _, err := w.Write(data); err != nil {
							http.Error(w, "Failed to write response", http.StatusInternalServerError)
						}
						return
					}
					appInstance.HandleJSONShorten(w, r)
				})
			} else {
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandlePostURL(w, r)
				})
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Method not allowed", http.StatusBadRequest)
				})
			}

			// Вызываем сервер
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")

			// Читаем тело ответа
			responseBody := rr.Body.Bytes()
			var responseString string

			// Если ожидается сжатый ответ, распаковываем его
			if tt.expectGzip {
				assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"), "Expected gzip Content-Encoding")
				gz, err := gzip.NewReader(bytes.NewReader(responseBody))
				assert.NoError(t, err, "Failed to create gzip reader")
				defer gz.Close()
				decompressed, err := io.ReadAll(gz)
				assert.NoError(t, err, "Failed to decompress response")
				responseString = string(decompressed)
			} else {
				responseString = string(responseBody)
			}

			if tt.expectedBody != "" {
				if tt.isJSON {
					assert.Contains(t, responseString, tt.expectedBody, "Expected JSON response with short URL")
				} else {
					assert.Equal(t, tt.expectedBody, responseString, "Expected exact response body")
				}
			}
			if tt.expectedStored {
				// Извлекаем ID из shortURL
				id := svc.ExtractIDFromShortURL(responseString)
				if tt.isJSON {
					var resp struct {
						Result string `json:"result"`
						Filler string `json:"filler,omitempty"`
					}
					err := json.Unmarshal([]byte(responseString), &resp)
					assert.NoError(t, err, "Failed to unmarshal JSON response")
					id = svc.ExtractIDFromShortURL(resp.Result)
				}
				_, exists := repo.Get(id)
				assert.True(t, exists, "Expected URL to be stored")
				if tt.expectedCode != http.StatusConflict {
					assert.Contains(t, responseString, cfg.BaseURL, "Expected short URL to contain BaseURL")
				}
			}
		})
	}
}

// Тесты для HandleGetURL
func TestHandleGetURL(t *testing.T) {
	// Создаём временный файл для тестов
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Создаём зависимости
	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test_secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	appInstance := NewApp(svc, nil, cfg)

	// Таблица тестов
	tests := []struct {
		name         string
		method       string
		path         string
		storeSetup   func()
		userID       string
		expectedCode int
		expectedBody string
		expectedLoc  string
	}{
		{
			name:   "Success",
			method: http.MethodGet,
			path:   "/testID",
			userID: "test_user",
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "test_user")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode: http.StatusTemporaryRedirect,
			expectedLoc:  "https://example.com",
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodPost,
			path:         "/testID",
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
		{
			name:         "NotFound",
			method:       http.MethodGet,
			path:         "/unknownID",
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: "URL not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			// Настраиваем хранилище
			tt.storeSetup()

			// Создаём маршрутизатор chi
			r := chi.NewRouter()
			r.Use(middleware.AuthMiddleware(svc, cfg, zap.NewNop()))
			r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				appInstance.HandleGetURL(w, r)
			})

			// Создаём тестовый сервер
			server := httptest.NewServer(r)
			defer server.Close()

			// Создаём запрос с UserID в контексте
			req, err := http.NewRequest(tt.method, strings.TrimSuffix(server.URL, "/")+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			ctx := context.Background()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, middleware.UserIDKey{}, tt.userID)
			}
			req = req.WithContext(ctx)
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // Не следовать редиректам
				},
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			// Читаем тело ответа
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, resp.StatusCode, "Status code mismatch")
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, string(body), "Body mismatch")
			}
			if tt.expectedLoc != "" {
				assert.Equal(t, tt.expectedLoc, resp.Header.Get("Location"), "Body mismatch")
			}
		})
	}
}

// Тесты для HandleJSONExpand
func TestHandleJSONExpand(t *testing.T) {
	// Создаём временный файл для тестов
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Создаём зависимости
	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test_secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	appInstance := NewApp(svc, nil, cfg)

	tests := []struct {
		name         string
		method       string
		path         string
		storeSetup   func()
		userID       string
		expectedCode int
		expectedBody string
	}{
		{
			name:   "Success",
			method: http.MethodGet,
			path:   "/api/expand/testID",
			userID: "test_user",
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "test_user")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode: http.StatusOK,
			expectedBody: `{"url":"https://example.com"}`,
		},
		{
			name:         "NotFound",
			method:       http.MethodGet,
			path:         "/api/expand/unknownID",
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"URL not found"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			// Настраиваем хранилище
			tt.storeSetup()
			r := chi.NewRouter()
			r.Use(middleware.AuthMiddleware(svc, cfg, zap.NewNop()))
			r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
				appInstance.HandleJSONExpand(w, r)
			})
			server := httptest.NewServer(r)
			defer server.Close()
			// Создаём запрос с UserID
			req, err := http.NewRequest(tt.method, strings.TrimSuffix(server.URL, "/")+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			ctx := context.Background()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, middleware.UserIDKey{}, tt.userID)
			}
			req = req.WithContext(ctx)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			assert.Equal(t, tt.expectedCode, resp.StatusCode, "Status code mismatch")
			assert.Equal(t, tt.expectedBody, string(body), "Body mismatch")
		})
	}
}

// Тесты для HandlePing
func TestHandlePing(t *testing.T) {
	// Создаём временный файл для тестов
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Создаём зависимости для svc
	cfg := &config.Config{
		JWTSecret: "test_secret",
	}
	repo, err := repository.NewFileRepository(tempFile.Name(), zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, "http://localhost:8080", cfg.JWTSecret)

	tests := []struct {
		name           string
		method         string
		dbSetup        func(*gomock.Controller) repository.Database
		userID         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful ping",
			method: http.MethodGet,
			userID: "test_user",
			dbSetup: func(ctrl *gomock.Controller) repository.Database {
				mockDB := repository.NewMockDatabase(ctrl)
				mockDB.EXPECT().Ping().Return(nil)
				return mockDB
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:   "database connection failed",
			method: http.MethodGet,
			userID: "test_user",
			dbSetup: func(ctrl *gomock.Controller) repository.Database {
				mockDB := repository.NewMockDatabase(ctrl)
				mockDB.EXPECT().Ping().Return(errors.New("connection failed"))
				return mockDB
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Database connection failed\n",
		},
		{
			name:   "no database configured",
			method: http.MethodGet,
			userID: "test_user",
			dbSetup: func(ctrl *gomock.Controller) repository.Database {
				return nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Database not configured\n",
		},
		{
			name:           "wrong method",
			method:         http.MethodPost,
			userID:         "test_user",
			dbSetup:        func(*gomock.Controller) repository.Database { return nil },
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём контроллер gomock для каждого подтеста
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Настраиваем мок или возвращаем nil
			db := tt.dbSetup(ctrl)

			// Создаём App с зависимостями
			appInstance := NewApp(svc, db, cfg)

			// Настраиваем маршрутизатор
			r := chi.NewRouter()
			r.Use(middleware.AuthMiddleware(svc, cfg, zap.NewNop()))
			r.Get("/ping", appInstance.HandlePing)

			// Создаём тестовый запрос
			req := httptest.NewRequest(tt.method, "/ping", nil)
			ctx := context.Background()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, middleware.UserIDKey{}, tt.userID)
			}
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			// Выполняем запрос
			r.ServeHTTP(w, req)

			// Проверяем статус и тело ответа
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

// Тесты для HandleBatchShorten
func TestHandleBatchShorten(t *testing.T) {
	// Создаём временный файл для тестов
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Создаём зависимости
	cfg := &config.Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test_secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	appInstance := NewApp(svc, nil, cfg)

	// Таблица тестов
	tests := []struct {
		name         string
		method       string
		body         io.Reader
		contentType  string
		useGzip      bool
		storeSetup   func()
		userID       string
		expectedCode int
		expectedBody string
		verifyStore  bool
	}{
		{
			name:         "Success",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"https://test.com"}]`),
			contentType:  "application/json",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusCreated,
			verifyStore:  true,
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodGet,
			body:         nil,
			contentType:  "application/json",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
			verifyStore:  false,
		},
		{
			name:         "InvalidContentType",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"1","original_url":"https://example.com"}]`),
			contentType:  "text/plain",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Content-Type must be application/json\n",
			verifyStore:  false,
		},
		{
			name:         "InvalidJSON",
			method:       http.MethodPost,
			body:         strings.NewReader(`{invalid json}`),
			contentType:  "application/json",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid JSON\n",
			verifyStore:  false,
		},
		{
			name:         "EmptyBatch",
			method:       http.MethodPost,
			body:         strings.NewReader(`[]`),
			contentType:  "application/json",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Empty batch\n",
			verifyStore:  false,
		},
		{
			name:         "MissingCorrelationID",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"","original_url":"https://example.com"}]`),
			contentType:  "application/json",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Missing correlation_id\n",
			verifyStore:  false,
		},
		{
			name:         "InvalidURL",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"1","original_url":"invalid-url"}]`),
			contentType:  "application/json",
			useGzip:      false,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid URL\n",
			verifyStore:  false,
		},
		{
			name:         "GzipRequestSuccess",
			method:       http.MethodPost,
			body:         nil,
			contentType:  "application/json",
			useGzip:      true,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusCreated,
			verifyStore:  true,
		},
		{
			name:         "NoUserID",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"1","original_url":"https://example.com"}]`),
			contentType:  "application/json",
			useGzip:      false,
			userID:       "",
			storeSetup:   func() {},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "Unauthorized\n",
			verifyStore:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			tt.storeSetup()

			// Подготавливаем тело запроса
			var requestBody = tt.body
			if tt.useGzip {
				data := `[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"https://test.com"}]`
				compressed, err := compressData([]byte(data))
				assert.NoError(t, err, "Failed to compress request body")
				requestBody = bytes.NewReader(compressed)
			}

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/api/shorten/batch", requestBody)
			req.Header.Set("Content-Type", tt.contentType)
			if tt.useGzip {
				req.Header.Set("Content-Encoding", "gzip")
			}
			ctx := context.Background()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, middleware.UserIDKey{}, tt.userID)
			}
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			// Настраиваем маршрутизатор
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
			r.Use(middleware.AuthMiddleware(svc, cfg, zap.NewNop()))
			r.Post("/api/shorten/batch", appInstance.HandleBatchShorten)

			// Выполняем запрос
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String(), "Response body mismatch")
			}
			if tt.verifyStore {
				var resp []models.BatchResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err, "Failed to unmarshal JSON response")
				assert.Len(t, resp, 2, "Expected two responses")
				for _, r := range resp {
					id := svc.ExtractIDFromShortURL(r.ShortURL)
					_, exists := repo.Get(id)
					assert.True(t, exists, "URL should be stored")
					assert.Contains(t, r.ShortURL, cfg.BaseURL, "Short URL should contain BaseURL")
				}
			}
		})
	}
}

// Тесты для HandleUserURLs
func TestHandleUserURLs(t *testing.T) {
	// Создаём временный файл для тестов
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Создаём зависимости
	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test_secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	appInstance := NewApp(svc, nil, cfg)

	tests := []struct {
		name         string
		method       string
		userID       string
		storeSetup   func()
		expectedCode int
		expectedBody string
	}{
		{
			name:   "SuccessWithURLs",
			method: http.MethodGet,
			userID: "test_user",
			storeSetup: func() {
				// Явно сохраняем URLs с userID
				_, err := repo.Save("id1", "https://example.com", "test_user")
				assert.NoError(t, err)
				_, err = repo.Save("id2", "https://test.com", "test_user")
				assert.NoError(t, err)
			},
			expectedCode: http.StatusOK,
			expectedBody: `[{"short_url":"http://localhost:8080/","original_url":"https://example.com"},{"short_url":"http://localhost:8080/","original_url":"https://test.com"}]`,
		},
		{
			name:         "NoURLs",
			method:       http.MethodGet,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusNoContent,
			expectedBody: "",
		},
		{
			name:         "NoUserID",
			method:       http.MethodGet,
			userID:       "",
			storeSetup:   func() {},
			expectedCode: http.StatusNoContent,
			expectedBody: "",
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodPost,
			userID:       "test_user",
			storeSetup:   func() {},
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			tt.storeSetup()

			// Настраиваем маршрутизатор
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
			r.Use(middleware.AuthMiddleware(svc, cfg, zap.NewNop()))
			r.Get("/api/user/urls", appInstance.HandleUserURLs)

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/api/user/urls", nil)
			ctx := context.Background()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, middleware.UserIDKey{}, tt.userID)
			}
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			// Выполняем запрос
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
			if tt.expectedBody != "" {
				if tt.expectedCode == http.StatusOK {
					var resp []models.ShortURLResponse
					err := json.Unmarshal(rr.Body.Bytes(), &resp)
					assert.NoError(t, err, "Failed to unmarshal JSON response")
					assert.Len(t, resp, 2, "Expected two URLs")
					for _, r := range resp {
						assert.Contains(t, r.ShortURL, cfg.BaseURL, "Short URL should contain BaseURL")
						assert.Contains(t, []string{"https://example.com", "https://test.com"}, r.OriginalURL, "Unexpected original URL")
					}
				} else {
					assert.Equal(t, tt.expectedBody, rr.Body.String(), "Response body mismatch")
				}
			}
		})
	}
}
