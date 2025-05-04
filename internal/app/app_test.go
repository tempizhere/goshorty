package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/config"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
	}
	repo, err := repository.NewFileRepository(cfg.FileStoragePath)
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, cfg.BaseURL)
	appInstance := NewApp(svc)

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
		expectedCode    int
		expectedBody    string
		expectedStored  bool
		expectGzip      bool
	}{
		{
			name:            "Success",
			method:          http.MethodPost,
			url:             "/",
			contentType:     "text/plain",
			body:            strings.NewReader("https://example.com"),
			isJSON:          false,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusCreated,
			expectedStored:  true,
			expectGzip:      false,
		},
		{
			name:            "InvalidMethod",
			method:          http.MethodGet,
			url:             "/",
			contentType:     "text/plain",
			body:            nil,
			isJSON:          false,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusBadRequest,
			expectedBody:    "Method not allowed\n",
			expectedStored:  false,
			expectGzip:      false,
		},
		{
			name:            "InvalidContentType",
			method:          http.MethodPost,
			url:             "/",
			contentType:     "application/json",
			body:            strings.NewReader("https://example.com"),
			isJSON:          false,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusCreated,
			expectedStored:  true,
			expectGzip:      false,
		},
		{
			name:            "EmptyBody",
			method:          http.MethodPost,
			url:             "/",
			contentType:     "text/plain",
			body:            strings.NewReader(""),
			isJSON:          false,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusBadRequest,
			expectedBody:    "empty URL\n",
			expectedStored:  false,
			expectGzip:      false,
		},
		{
			name:            "ReadBodyError",
			method:          http.MethodPost,
			url:             "/",
			contentType:     "text/plain",
			body:            strings.NewReader("https://example.com"),
			isJSON:          false,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusBadRequest,
			expectedBody:    "Failed to read request body\n",
			expectedStored:  false,
			expectGzip:      false,
		},
		{
			name:            "JSONSuccess",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:          true,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusCreated,
			expectedBody:    `{"result":"` + cfg.BaseURL + "/",
			expectedStored:  true,
			expectGzip:      false,
		},
		{
			name:            "JSONInvalid",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{invalid json}`),
			isJSON:          true,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusBadRequest,
			expectedBody:    "Invalid JSON\n",
			expectGzip:      false,
		},
		{
			name:            "JSONEmptyURL",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{"url":""}`),
			isJSON:          true,
			useGzipRequest:  false,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusBadRequest,
			expectedBody:    "empty URL\n",
			expectGzip:      false,
		},
		{
			name:            "GzipRequestJSONSuccess",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            nil, // Будет установлено в тесте
			isJSON:          true,
			useGzipRequest:  true,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusCreated,
			expectedBody:    `{"result":"` + cfg.BaseURL + "/",
			expectedStored:  true,
			expectGzip:      false,
		},
		{
			name:            "GzipRequestTextSuccess",
			method:          http.MethodPost,
			url:             "/",
			contentType:     "application/x-gzip",
			body:            nil, // Будет установлено в тесте
			isJSON:          false,
			useGzipRequest:  true,
			useGzipResponse: false,
			largeResponse:   false,
			expectedCode:    http.StatusCreated,
			expectedStored:  true,
			expectGzip:      false,
		},
		{
			name:            "GzipResponseJSONSuccessLarge",
			method:          http.MethodPost,
			url:             "/api/shorten",
			contentType:     "application/json",
			body:            strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:          true,
			useGzipRequest:  false,
			useGzipResponse: true,
			largeResponse:   true,
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
			useGzipRequest:  false,
			useGzipResponse: true,
			largeResponse:   false,
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
			useGzipRequest:  false,
			useGzipResponse: true,
			largeResponse:   true,
			expectedCode:    http.StatusCreated,
			expectedStored:  true,
			expectGzip:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем хранилище
			repo.Clear()

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
			rr := httptest.NewRecorder()

			// Для ReadBodyError подменяем тело запроса
			if tt.name == "ReadBodyError" {
				req.Body = io.NopCloser(&errorReader{})
			}

			// Создаём маршрутизатор с GzipMiddleware
			r := chi.NewRouter()
			r.Use(middleware.GzipMiddleware)
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
						shortURL, err := appInstance.createShortURL(reqBody.URL)
						if err != nil {
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
							return
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
				assert.Contains(t, responseString, cfg.BaseURL, "Expected short URL to contain BaseURL")
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
	repo, err := repository.NewFileRepository(tempFile.Name())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, "http://localhost:8080")
	appInstance := NewApp(svc)

	// Таблица тестов
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
				err := repo.Save("testID", "https://example.com")
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
	repo, err := repository.NewFileRepository(tempFile.Name())
	assert.NoError(t, err, "Failed to create file repository")
	svc := service.NewService(repo, "http://localhost:8080")
	appInstance := NewApp(svc)

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
				err := repo.Save("testID", "https://example.com")
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
