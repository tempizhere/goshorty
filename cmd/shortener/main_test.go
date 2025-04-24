package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/cmd/shortener/config"
)

// errorReader симулирует ошибку чтения
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// Тесты для handlePostURL
func TestHandlePostURL(t *testing.T) {
	// Таблица тестов
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           io.Reader
		expectedCode   int
		expectedBody   string
		expectedStored bool
	}{
		{
			name:           "Success",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           strings.NewReader("https://example.com"),
			expectedCode:   http.StatusCreated,
			expectedBody:   "http://localhost:8080/aHR0cHM6",
			expectedStored: true,
		},
		{
			name:         "InvalidMethod",
			method:       http.MethodGet,
			contentType:  "text/plain",
			body:         nil,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Method not allowed\n",
		},
		{
			name:         "InvalidContentType",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         strings.NewReader("https://example.com"),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Content-Type must be text/plain\n",
		},
		{
			name:         "EmptyBody",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         strings.NewReader(""),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Empty URL\n",
		},
		{
			name:         "ReadBodyError",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body: strings.NewReader("https://example.com"),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to read request body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем urlStore
			urlStore = make(map[string]string)

			// Создаём тестовую конфигурацию
			cfg := &config.Config{
				RunAddr: ":8080",
				BaseURL: "http://localhost:8080",
			}

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/", tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			// Для ReadBodyError подменяем тело запроса
			if tt.name == "ReadBodyError" {
				req.Body = io.NopCloser(&errorReader{})
			}

			// Вызываем обработчик
			handlePostURL(rr, req, cfg)

			// Проверяем результаты
			assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String(), "Body mismatch")
			}
			if tt.expectedStored {
				assert.NotEmpty(t, urlStore, "Expected URL to be stored")
				assert.Contains(t, rr.Body.String(), tt.expectedBody, "Expected short URL")
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
			expectedBody: "", // Пустое тело
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

func TestHandleJSONShorten(t *testing.T) {
    // Таблица тестов
    tests := []struct {
        name         string
        method       string
        contentType  string
        body         string
        expectedCode int
        expectedBody string
        expectedErr  string
    }{
        {
            name:        "Success",
            method:      http.MethodPost,
            contentType: "application/json",
            body:        `{"url": "https://example.com"}`,
            expectedCode: http.StatusCreated,
            expectedBody: `{"result":"http://localhost:8080/aHR0cHM6"}`,
        },
        {
            name:         "InvalidMethod",
            method:       http.MethodGet,
            contentType:  "application/json",
            body:         `{"url": "https://example.com"}`,
            expectedCode: http.StatusMethodNotAllowed,
            expectedErr:  "Method not allowed\n",
        },
        {
            name:         "InvalidContentType",
            method:       http.MethodPost,
            contentType:  "text/plain",
            body:         `{"url": "https://example.com"}`,
            expectedCode: http.StatusBadRequest,
            expectedErr:  "Content-Type must be application/json\n",
        },
        {
            name:         "EmptyURL",
            method:       http.MethodPost,
            contentType:  "application/json",
            body:         `{"url": ""}`,
            expectedCode: http.StatusBadRequest,
            expectedErr:  "URL is required\n",
        },
        {
            name:         "InvalidJSON",
            method:       http.MethodPost,
            contentType:  "application/json",
            body:         `{"url": "https://example.com"`, // Некорректный JSON
            expectedCode: http.StatusBadRequest,
            expectedErr:  "Failed to decode JSON\n",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Очищаем urlStore
            urlStore = make(map[string]string)

            // Создаём тестовую конфигурацию
            cfg := &config.Config{
                RunAddr: ":8080",
                BaseURL: "http://localhost:8080",
            }

            // Создаём запрос
            req := httptest.NewRequest(tt.method, "/api/shorten", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", tt.contentType)
            rr := httptest.NewRecorder()

            // Вызываем обработчик
            handleJSONShorten(rr, req, cfg)

            // Проверяем результаты
            assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
            if tt.expectedErr != "" {
                assert.Equal(t, tt.expectedErr, rr.Body.String(), "Error message mismatch")
            } else {
                assert.Equal(t, tt.expectedBody, strings.TrimSpace(rr.Body.String()), "Body mismatch")
            }
        })
    }
}

func TestHandleJSONExpand(t *testing.T) {
    // Таблица тестов
    tests := []struct {
        name         string
        method       string
        query        string
        storeSetup   func()
        expectedCode int
        expectedLoc  string
        expectedErr  string
    }{
        {
            name:   "Success",
            method: http.MethodGet,
            query:  "?id=testID",
            storeSetup: func() {
                urlStore["testID"] = "https://example.com"
            },
            expectedCode: http.StatusTemporaryRedirect,
            expectedLoc:  "https://example.com",
        },
        {
            name:         "InvalidMethod",
            method:       http.MethodPost,
            query:        "?id=testID",
            storeSetup:   func() {},
            expectedCode: http.StatusMethodNotAllowed,
            expectedErr:  "Method not allowed\n",
        },
        {
            name:         "MissingID",
            method:       http.MethodGet,
            query:        "",
            storeSetup:   func() {},
            expectedCode: http.StatusBadRequest,
            expectedErr:  "ID parameter is required\n",
        },
        {
            name:         "NotFound",
            method:       http.MethodGet,
            query:        "?id=unknownID",
            storeSetup:   func() {},
            expectedCode: http.StatusNotFound,
            expectedErr:  "URL not found\n",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Очищаем urlStore
            urlStore = make(map[string]string)
            // Настраиваем urlStore
            tt.storeSetup()

            // Создаём запрос
            req := httptest.NewRequest(tt.method, "/api/expand"+tt.query, nil)
            rr := httptest.NewRecorder()

            // Вызываем обработчик напрямую
            handleJSONExpand(rr, req)

            // Проверяем результаты
            assert.Equal(t, tt.expectedCode, rr.Code, "Status code mismatch")
            if tt.expectedErr != "" {
                assert.Equal(t, tt.expectedErr, rr.Body.String(), "Error message mismatch")
            }
            if tt.expectedLoc != "" {
                assert.Equal(t, tt.expectedLoc, rr.Header().Get("Location"), "Location header mismatch")
            }
        })
    }
}