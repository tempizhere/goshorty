package app

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

// setupTestEnvironment создаёт тестовое окружение с временным файлом и зависимостями
func setupTestEnvironment(t *testing.T) (*config.Config, repository.Repository, *service.Service, *App, *zap.Logger, func()) {
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")

	cleanup := func() {
		_ = os.Remove(tempFile.Name())
	}

	cfg := &config.Config{
		RunAddr:         ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test-secret",
	}

	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")

	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	logger := zap.NewNop()
	appInstance := NewApp(svc, nil, logger)

	return cfg, repo, svc, appInstance, logger, cleanup
}

// createTestRequest создаёт тестовый HTTP запрос с заданными параметрами
func createTestRequest(method, url, contentType string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", contentType)
	return req
}

// createTestRouter создаёт маршрутизатор с необходимыми middleware
func createTestRouter(svc *service.Service, logger *zap.Logger, routes map[string]http.HandlerFunc) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))

	for pattern, handler := range routes {
		r.HandleFunc(pattern, handler)
	}

	return r
}

// createTestRouterWithGzip создаёт маршрутизатор с Gzip middleware
func createTestRouterWithGzip(svc *service.Service, logger *zap.Logger, routes map[string]http.HandlerFunc) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.AuthMiddleware(svc, logger))

	for pattern, handler := range routes {
		r.HandleFunc(pattern, handler)
	}

	return r
}

// assertResponseCode проверяет код ответа
func assertResponseCode(t *testing.T, rr *httptest.ResponseRecorder, expectedCode int) {
	assert.Equal(t, expectedCode, rr.Code, "Status code mismatch")
}

// assertResponseBody проверяет тело ответа
func assertResponseBody(t *testing.T, rr *httptest.ResponseRecorder, expectedBody string, isJSON bool) {
	if expectedBody != "" {
		if isJSON {
			assert.Contains(t, rr.Body.String(), expectedBody, "Expected JSON response with short URL")
		} else {
			assert.Equal(t, expectedBody, rr.Body.String(), "Expected exact response body")
		}
	}
}

// assertURLStored проверяет, что URL сохранен в хранилище
func assertURLStored(t *testing.T, repo repository.Repository, responseBody, baseURL string, expectedCode int, isJSON bool) {
	if expectedCode == http.StatusCreated || expectedCode == http.StatusConflict {
		var id string
		if isJSON {
			var resp struct {
				Result string `json:"result"`
			}
			err := json.Unmarshal([]byte(responseBody), &resp)
			assert.NoError(t, err, "Failed to unmarshal JSON response")
			id = resp.Result[strings.LastIndex(resp.Result, "/")+1:]
		} else {
			id = responseBody[strings.LastIndex(responseBody, "/")+1:]
		}

		_, exists := repo.Get(id)
		assert.True(t, exists, "Expected URL to be stored")
		if expectedCode != http.StatusConflict {
			assert.Contains(t, responseBody, baseURL, "Expected short URL to contain BaseURL")
		}
	}
}

// assertBatchResponse проверяет пакетный ответ
func assertBatchResponse(t *testing.T, rr *httptest.ResponseRecorder, repo repository.Repository, baseURL string, expectedCount int) {
	var resp []models.BatchResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err, "Failed to unmarshal JSON response")
	assert.Len(t, resp, expectedCount, "Expected correct number of responses")

	for _, r := range resp {
		id := r.ShortURL[strings.LastIndex(r.ShortURL, "/")+1:]
		_, exists := repo.Get(id)
		assert.True(t, exists, "URL should be stored")
		assert.Contains(t, r.ShortURL, baseURL, "Short URL should contain BaseURL")
	}
}

// TestHandlePostURL тестирует обработку POST запросов для создания коротких URL
func TestHandlePostURL(t *testing.T) {
	cfg, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		url            string
		contentType    string
		body           io.Reader
		storeSetup     func()
		expectedCode   int
		expectedBody   string
		expectedStored bool
	}{
		{
			name:           "Success",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "text/plain",
			body:           strings.NewReader("https://example.com"),
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedStored: true,
		},
		{
			name:        "Duplicate URL",
			method:      http.MethodPost,
			url:         "/",
			contentType: "text/plain",
			body:        strings.NewReader("https://example.com"),
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "testUser")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode:   http.StatusConflict,
			expectedStored: true,
			expectedBody:   "http://localhost:8080/testID",
		},
		{
			name:           "InvalidMethod",
			method:         http.MethodGet,
			url:            "/",
			contentType:    "text/plain",
			body:           nil,
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Method not allowed\n",
			expectedStored: false,
		},
		{
			name:           "EmptyBody",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "text/plain",
			body:           strings.NewReader(""),
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "empty URL\n",
			expectedStored: false,
		},
		{
			name:           "ReadBodyError",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "text/plain",
			body:           strings.NewReader("https://example.com"),
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Failed to read request body\n",
			expectedStored: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			tt.storeSetup()

			// Создаём запрос
			req := createTestRequest(tt.method, tt.url, tt.contentType, tt.body)
			rr := httptest.NewRecorder()

			// Для ReadBodyError подменяем тело запроса
			if tt.name == "ReadBodyError" {
				req.Body = io.NopCloser(&errorReader{})
			}

			// Создаём маршрутизатор
			routes := map[string]http.HandlerFunc{
				"/": appInstance.HandlePostURL,
			}
			r := createTestRouter(svc, logger, routes)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Method not allowed", http.StatusBadRequest)
			})

			// Вызываем сервер
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assertResponseCode(t, rr, tt.expectedCode)
			assertResponseBody(t, rr, tt.expectedBody, false)
			if tt.expectedStored {
				assertURLStored(t, repo, rr.Body.String(), cfg.BaseURL, tt.expectedCode, false)
			}
		})
	}
}

// TestHandleJSONShorten тестирует обработку JSON запросов для создания коротких URL
func TestHandleJSONShorten(t *testing.T) {
	cfg, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		url            string
		contentType    string
		body           io.Reader
		storeSetup     func()
		expectedCode   int
		expectedBody   string
		expectedStored bool
	}{
		{
			name:           "JSONSuccess",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           strings.NewReader(`{"url":"https://example.com"}`),
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedBody:   `{"result":"` + cfg.BaseURL + "/",
			expectedStored: true,
		},
		{
			name:        "JSONDuplicateURL",
			method:      http.MethodPost,
			url:         "/api/shorten",
			contentType: "application/json",
			body:        strings.NewReader(`{"url":"https://example.com"}`),
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "testUser")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode:   http.StatusConflict,
			expectedBody:   `{"result":"http://localhost:8080/testID"}`,
			expectedStored: true,
		},
		{
			name:           "JSONInvalid",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           strings.NewReader(`{invalid json}`),
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
			expectedStored: false,
		},
		{
			name:           "JSONEmptyURL",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           strings.NewReader(`{"url":""}`),
			storeSetup:     func() {},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "empty URL\n",
			expectedStored: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()
			tt.storeSetup()

			// Создаём запрос и маршрутизатор
			req := createTestRequest(tt.method, tt.url, tt.contentType, tt.body)
			rr := httptest.NewRecorder()

			routes := map[string]http.HandlerFunc{
				"/api/shorten": appInstance.HandleJSONShorten,
			}
			r := createTestRouter(svc, logger, routes)

			// Вызываем сервер
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assertResponseCode(t, rr, tt.expectedCode)
			assertResponseBody(t, rr, tt.expectedBody, true)
			if tt.expectedStored {
				assertURLStored(t, repo, rr.Body.String(), cfg.BaseURL, tt.expectedCode, true)
			}
		})
	}
}

// TestHandleGzipRequests тестирует обработку запросов с Gzip сжатием
func TestHandleGzipRequests(t *testing.T) {
	cfg, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		url            string
		contentType    string
		body           io.Reader
		useGzipRequest bool
		storeSetup     func()
		expectedCode   int
		expectedStored bool
	}{
		{
			name:           "GzipRequestJSONSuccess",
			method:         http.MethodPost,
			url:            "/api/shorten",
			contentType:    "application/json",
			body:           nil, // Будет установлено в тесте
			useGzipRequest: true,
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedStored: true,
		},
		{
			name:           "GzipRequestTextSuccess",
			method:         http.MethodPost,
			url:            "/",
			contentType:    "application/x-gzip",
			body:           nil, // Будет установлено в тесте
			useGzipRequest: true,
			storeSetup:     func() {},
			expectedCode:   http.StatusCreated,
			expectedStored: true,
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
				if !strings.Contains(tt.contentType, "json") {
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
			rr := httptest.NewRecorder()

			// Создаём маршрутизатор с GzipMiddleware и AuthMiddleware
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
			r.Use(middleware.AuthMiddleware(svc, logger))

			if strings.Contains(tt.contentType, "json") {
				r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandleJSONShorten(w, r)
				})
			} else {
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandlePostURL(w, r)
				})
			}

			// Вызываем сервер
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
			if tt.expectedStored {
				assert.Contains(t, rr.Body.String(), cfg.BaseURL, "Expected short URL to contain BaseURL")
			}
		})
	}
}

// TestHandleGzipResponses тестирует обработку ответов с Gzip сжатием
func TestHandleGzipResponses(t *testing.T) {
	cfg, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name            string
		method          string
		url             string
		contentType     string
		body            io.Reader
		useGzipResponse bool
		largeResponse   bool
		storeSetup      func()
		expectedCode    int
		expectedBody    string
		expectedStored  bool
		expectGzip      bool
	}{
		{
			name:            "GzipResponseJSONSuccessLarge",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{"url":"https://example.com"}`),
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

			// Создаём запрос
			req := httptest.NewRequest(tt.method, tt.url, tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			if tt.useGzipResponse {
				req.Header.Set("Accept-Encoding", "gzip")
			}
			rr := httptest.NewRecorder()

			// Создаём маршрутизатор с GzipMiddleware и AuthMiddleware
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
			r.Use(middleware.AuthMiddleware(svc, logger))

			if tt.largeResponse {
				r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
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

					userID, ok := middleware.GetUserID(r)
					if !ok {
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
						return
					}

					shortURL, err := appInstance.createShortURL(reqBody.URL, userID)
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
						Filler: strings.Repeat("x", 1400), // Наполнитель для размера > 1400 байт
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
				})
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandlePostURL(w, r)
				})
			} else {
				r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandleJSONShorten(w, r)
				})
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandlePostURL(w, r)
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
				defer func() {
					if err := gz.Close(); err != nil {
						t.Logf("Failed to close gzip reader: %v", err)
					}
				}()
				decompressed, err := io.ReadAll(gz)
				assert.NoError(t, err, "Failed to decompress response")
				responseString = string(decompressed)
			} else {
				responseString = string(responseBody)
			}

			if tt.expectedBody != "" {
				assert.Contains(t, responseString, tt.expectedBody, "Expected JSON response with short URL")
			}
			if tt.expectedStored {
				// Извлекаем ID из shortURL
				if tt.contentType == "application/json" {
					var resp struct {
						Result string `json:"result"`
						Filler string `json:"filler,omitempty"`
					}
					err := json.Unmarshal([]byte(responseString), &resp)
					assert.NoError(t, err, "Failed to unmarshal JSON response")
					id := resp.Result[strings.LastIndex(resp.Result, "/")+1:]
					_, exists := repo.Get(id)
					assert.True(t, exists, "Expected URL to be stored")
					if tt.expectedCode != http.StatusConflict {
						assert.Contains(t, responseString, cfg.BaseURL, "Expected short URL to contain BaseURL")
					}
				} else {
					// Для text/plain ответа извлекаем ID напрямую
					id := responseString[strings.LastIndex(responseString, "/")+1:]
					_, exists := repo.Get(id)
					assert.True(t, exists, "Expected URL to be stored")
					assert.Contains(t, responseString, cfg.BaseURL, "Expected short URL to contain BaseURL")
				}
			}
		})
	}
}

// TestHandleGetURL тестирует обработку GET запросов для получения оригинальных URL
func TestHandleGetURL(t *testing.T) {
	_, repo, _, appInstance, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name         string
		method       string
		path         string
		storeSetup   func()
		expectedCode int
		expectedBody string
		expectedLoc  string
	}{
		{
			name:   "Success",
			method: http.MethodGet,
			path:   "/testID",
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "testUser")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode: http.StatusTemporaryRedirect,
			expectedLoc:  "https://example.com",
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodPost,
			path:         "/testID",
			storeSetup:   func() {},
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
		{
			name:         "NotFound",
			method:       http.MethodGet,
			path:         "/unknownID",
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
			r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				appInstance.HandleGetURL(w, r)
			})

			// Создаём тестовый сервер
			server := httptest.NewServer(r)
			defer server.Close()

			// Отправляем запрос
			req, err := http.NewRequest(tt.method, strings.TrimSuffix(server.URL, "/")+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // Не следовать редиректам
				},
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

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

// TestHandleJSONExpand тестирует обработку JSON запросов для получения оригинальных URL
func TestHandleJSONExpand(t *testing.T) {
	_, repo, _, appInstance, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name         string
		method       string
		path         string
		storeSetup   func()
		expectedCode int
		expectedBody string
	}{
		{
			name:   "Success",
			method: http.MethodGet,
			path:   "/api/expand/testID",
			storeSetup: func() {
				_, err := repo.Save("testID", "https://example.com", "testUser")
				assert.NoError(t, err, "Failed to save URL in storeSetup")
			},
			expectedCode: http.StatusOK,
			expectedBody: `{"url":"https://example.com"}`,
		},
		{
			name:         "NotFound",
			method:       http.MethodGet,
			path:         "/api/expand/unknownID",
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
			r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
				appInstance.HandleJSONExpand(w, r)
			})
			server := httptest.NewServer(r)
			defer server.Close()
			// Нормализуем URL, чтобы избежать двойных слэшей
			req, err := http.NewRequest(tt.method, strings.TrimSuffix(server.URL, "/")+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			assert.Equal(t, tt.expectedCode, resp.StatusCode, "Status code mismatch")
			assert.Equal(t, tt.expectedBody, string(body), "Body mismatch")
		})
	}
}

// mockDatabase - простой мок для Database интерфейса
type mockDatabase struct {
	pingErr error
}

func (m *mockDatabase) Ping() error {
	return m.pingErr
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *mockDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *mockDatabase) Begin() (*sql.Tx, error) {
	return nil, nil
}

// TestHandlePing тестирует обработку ping запросов для проверки состояния базы данных
func TestHandlePing(t *testing.T) {

	tests := []struct {
		name           string
		method         string
		dbSetup        func() repository.Database
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful ping",
			method: http.MethodGet,
			dbSetup: func() repository.Database {
				return &mockDatabase{pingErr: nil}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:   "database connection failed",
			method: http.MethodGet,
			dbSetup: func() repository.Database {
				return &mockDatabase{pingErr: errors.New("connection failed")}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Database connection failed\n",
		},
		{
			name:   "no database configured",
			method: http.MethodGet,
			dbSetup: func() repository.Database {
				return nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Database not configured\n",
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			dbSetup: func() repository.Database {
				return nil
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем мок или возвращаем nil
			db := tt.dbSetup()

			// Создаём App с зависимостями
			logger := zap.NewNop()
			appInstance := NewApp(nil, db, logger)

			// Настраиваем маршрутизатор
			r := chi.NewRouter()
			r.Get("/ping", appInstance.HandlePing)

			req := httptest.NewRequest(tt.method, "/ping", nil)
			w := httptest.NewRecorder()

			// Выполняем запрос
			r.ServeHTTP(w, req)

			// Проверяем статус и тело ответа
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

// TestHandleBatchShortenSuccess тестирует успешную обработку пакетных запросов
func TestHandleBatchShortenSuccess(t *testing.T) {
	cfg, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Очищаем хранилище
	repo.Clear()

	// Создаём запрос
	req := createTestRequest(http.MethodPost, "/api/shorten/batch", "application/json",
		strings.NewReader(`[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"https://test.com"}]`))
	rr := httptest.NewRecorder()

	// Настраиваем маршрутизатор
	routes := map[string]http.HandlerFunc{
		"/api/shorten/batch": appInstance.HandleBatchShorten,
	}
	r := createTestRouterWithGzip(svc, logger, routes)

	// Выполняем запрос
	r.ServeHTTP(rr, req)

	// Проверяем результаты
	assertResponseCode(t, rr, http.StatusCreated)
	assertBatchResponse(t, rr, repo, cfg.BaseURL, 2)
}

// TestHandleBatchShortenValidation тестирует валидацию пакетных запросов
func TestHandleBatchShortenValidation(t *testing.T) {
	_, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name         string
		method       string
		body         io.Reader
		contentType  string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "InvalidMethod",
			method:       http.MethodGet,
			body:         nil,
			contentType:  "application/json",
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
		{
			name:         "InvalidContentType",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"1","original_url":"https://example.com"}]`),
			contentType:  "text/plain",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Content-Type must be application/json\n",
		},
		{
			name:         "InvalidJSON",
			method:       http.MethodPost,
			body:         strings.NewReader(`{invalid json}`),
			contentType:  "application/json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid JSON\n",
		},
		{
			name:         "EmptyBatch",
			method:       http.MethodPost,
			body:         strings.NewReader(`[]`),
			contentType:  "application/json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Empty batch\n",
		},
		{
			name:         "MissingCorrelationID",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"","original_url":"https://example.com"}]`),
			contentType:  "application/json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Missing correlation_id\n",
		},
		{
			name:         "InvalidURL",
			method:       http.MethodPost,
			body:         strings.NewReader(`[{"correlation_id":"1","original_url":"invalid-url"}]`),
			contentType:  "application/json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid URL\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/api/shorten/batch", tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			// Настраиваем маршрутизатор
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
			r.Use(middleware.AuthMiddleware(svc, logger))
			r.Post("/api/shorten/batch", appInstance.HandleBatchShorten)

			// Выполняем запрос
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String(), "Response body mismatch")
			}
		})
	}
}

// TestHandleBatchShortenGzip тестирует обработку пакетных запросов с Gzip сжатием
func TestHandleBatchShortenGzip(t *testing.T) {
	cfg, repo, svc, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Очищаем хранилище
	repo.Clear()

	// Подготавливаем сжатое тело запроса
	data := `[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"https://test.com"}]`
	compressed, err := compressData([]byte(data))
	assert.NoError(t, err, "Failed to compress request body")
	requestBody := bytes.NewReader(compressed)

	// Создаём запрос
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", requestBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()

	// Настраиваем маршрутизатор
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Post("/api/shorten/batch", appInstance.HandleBatchShorten)

	// Выполняем запрос
	r.ServeHTTP(rr, req)

	// Проверяем результаты
	assert.Equal(t, http.StatusCreated, rr.Code, "Status code mismatch")

	var resp []models.BatchResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err, "Failed to unmarshal JSON response")
	assert.Len(t, resp, 2, "Expected two responses")

	for _, r := range resp {
		id := r.ShortURL[strings.LastIndex(r.ShortURL, "/")+1:]
		_, exists := repo.Get(id)
		assert.True(t, exists, "URL should be stored")
		assert.Contains(t, r.ShortURL, cfg.BaseURL, "Short URL should contain BaseURL")
	}
}

// TestHandleBatchDeleteURLsSuccess тестирует успешную обработку пакетных запросов для удаления URL
func TestHandleBatchDeleteURLsSuccess(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temporary file: %v", err)
		}
	}()

	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test-secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	logger := zap.NewNop()
	appInstance := NewApp(svc, nil, logger)

	// Очищаем хранилище
	repo.Clear()

	// Создаём запрос
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls",
		strings.NewReader(`["testID1","testID2"]`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Настраиваем маршрутизатор
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(svc, logger))
	r.Delete("/api/user/urls", appInstance.HandleBatchDeleteURLs)

	// Выполняем запрос
	r.ServeHTTP(rr, req)

	// Проверяем результаты
	assert.Equal(t, http.StatusAccepted, rr.Code, "Status code mismatch")
}

// TestHandleBatchDeleteURLsValidation тестирует валидацию пакетных запросов для удаления URL
func TestHandleBatchDeleteURLsValidation(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_storage_*.json")
	assert.NoError(t, err, "Failed to create temp file")
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temporary file: %v", err)
		}
	}()

	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: tempFile.Name(),
		JWTSecret:       "test-secret",
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath, zap.NewNop())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	logger := zap.NewNop()
	appInstance := NewApp(svc, nil, logger)

	tests := []struct {
		name         string
		method       string
		body         io.Reader
		contentType  string
		expectedCode int
	}{
		{
			name:         "WrongMethod",
			method:       http.MethodGet,
			body:         nil,
			contentType:  "application/json",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "InvalidContentType",
			method:       http.MethodDelete,
			body:         strings.NewReader(`["testID1"]`),
			contentType:  "text/plain",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "InvalidJSON",
			method:       http.MethodDelete,
			body:         strings.NewReader(`{invalid json}`),
			contentType:  "application/json",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/api/user/urls", tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			// Настраиваем маршрутизатор
			r := chi.NewRouter()
			r.Use(middleware.AuthMiddleware(svc, logger))
			r.Delete("/api/user/urls", appInstance.HandleBatchDeleteURLs)

			// Выполняем запрос
			r.ServeHTTP(rr, req)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
		})
	}
}
