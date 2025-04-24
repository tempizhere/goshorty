package main

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/cmd/shortener/config"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// errorReader симулирует ошибку чтения
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// Тесты для handlePostURL
// Тесты для handlePostURL и handleJSONShorten
// Тесты для handlePostURL и handleJSONShorten
func TestHandlePostURL(t *testing.T) {
	// Создаём конфигурацию
	cfg := &config.Config{
		RunAddr: ":8080",
		BaseURL: "http://localhost:8080",
	}

	// Таблица тестов
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           io.Reader
		isJSON         bool
		expectedCode   int
		expectedBody   string
		expectedStored bool
	}{
		{
			name:           "Success",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           strings.NewReader("https://example.com"),
			isJSON:         false,
			expectedCode:   http.StatusCreated,
			expectedStored: true,
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodGet,
			contentType:  "text/plain",
			body:         nil,
			isJSON:       false,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Method not allowed\n",
		},
		{
			name:         "InvalidContentType",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         strings.NewReader("https://example.com"),
			isJSON:       false,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Content-Type must be text/plain\n",
		},
		{
			name:         "EmptyBody",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         strings.NewReader(""),
			isJSON:       false,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Empty URL\n",
		},
		{
			name:         "ReadBodyError",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         strings.NewReader("https://example.com"),
			isJSON:       false,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to read request body\n",
		},
		{
			name:           "JSONSuccess",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           strings.NewReader(`{"url":"https://example.com"}`),
			isJSON:         true,
			expectedCode:   http.StatusCreated,
			expectedBody:   `{"result":"` + cfg.BaseURL + "/",
			expectedStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем urlStore (придуманное имя)
			urlStore = make(map[string]string)

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/", tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			// Для ReadBodyError подменяем тело запроса
			if tt.name == "ReadBodyError" {
				req.Body = io.NopCloser(&errorReader{})
			}

			// Вызываем обработчик
			if tt.isJSON {
				handleJSONShorten(rr, req, cfg)
			} else {
				handlePostURL(rr, req, cfg)
			}

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
			if tt.expectedBody != "" {
				if tt.isJSON {
					assert.Contains(t, rr.Body.String(), tt.expectedBody, "Expected JSON response with short URL")
				} else {
					assert.Equal(t, tt.expectedBody, rr.Body.String(), "Body mismatch")
				}
			}
			if tt.expectedStored {
				assert.NotEmpty(t, urlStore, "Expected URL to be stored")
				assert.Contains(t, rr.Body.String(), cfg.BaseURL, "Expected short URL to contain BaseURL")
			}
		})
	}
}

// Тесты для handleGetURL
func TestHandleGetURL(t *testing.T) {
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
				urlStore["testID"] = "https://example.com"
			},
			expectedCode: http.StatusTemporaryRedirect,
			expectedLoc:  "https://example.com",
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodPost,
			path:         "/testID",
			storeSetup:   func() {},
			expectedCode: http.StatusMethodNotAllowed, // 405
			expectedBody: "",                          // Пустое тело
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
			// Очищаем urlStore
			urlStore = make(map[string]string)
			// Настраиваем urlStore
			tt.storeSetup()

			// Создаём маршрутизатор chi
			r := chi.NewRouter()
			r.Get("/{id}", handleGetURL)

			// Создаём тестовый сервер
			server := httptest.NewServer(r)
			defer server.Close()

			// Создаём клиент
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // Не следовать редиректам
				},
			}

			// Отправляем запрос
			req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
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
				assert.Equal(t, tt.expectedLoc, resp.Header.Get("Location"), "Location header mismatch")
			}
		})
	}
}
func TestHandleJSONExpand(t *testing.T) {
	cfg := &config.Config{
		RunAddr: ":8080",
		BaseURL: "http://localhost:8080",
	}
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
				urlStore["testID"] = "https://example.com"
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
			urlStore = make(map[string]string)
			tt.storeSetup()
			r := chi.NewRouter()
			r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
				handleJSONExpand(w, r, cfg)
			})
			server := httptest.NewServer(r)
			defer server.Close()
			client := &http.Client{}
			req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
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
