package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/middleware"
)

func TestApp_HandleStats(t *testing.T) {
	// Создаем тестовые зависимости
	_, repo, _, appInstance, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("GET request with valid data", func(t *testing.T) {
		// Добавляем тестовые данные
		_, err := repo.Save("id1", "https://example1.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id2", "https://example2.com", "user1")
		assert.NoError(t, err)
		_, err = repo.Save("id3", "https://example3.com", "user2")
		assert.NoError(t, err)

		// Создаем запрос
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		appInstance.HandleStats(rr, req)

		// Проверяем результат
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), `"urls":3`)
		assert.Contains(t, rr.Body.String(), `"users":2`)
	})

	t.Run("POST request - should return Method Not Allowed", func(t *testing.T) {
		// Создаем запрос
		req := httptest.NewRequest(http.MethodPost, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		appInstance.HandleStats(rr, req)

		// Проверяем результат
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		assert.Equal(t, "Method not allowed\n", rr.Body.String())
	})

	t.Run("Empty repository", func(t *testing.T) {
		// Очищаем репозиторий
		repo.Clear()

		// Создаем запрос
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		appInstance.HandleStats(rr, req)

		// Проверяем результат
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), `"urls":0`)
		assert.Contains(t, rr.Body.String(), `"users":0`)
	})
}

func TestApp_HandleStats_WithMiddleware(t *testing.T) {
	// Создаем тестовые зависимости
	_, repo, _, appInstance, logger, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Добавляем тестовые данные
	_, err := repo.Save("id1", "https://example1.com", "user1")
	assert.NoError(t, err)

	tests := []struct {
		name           string
		trustedSubnet  string
		clientIP       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Empty trusted subnet - should deny access",
			trustedSubnet:  "",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied\n",
		},
		{
			name:           "Missing X-Real-IP header - should deny access",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied\n",
		},
		{
			name:           "IP not in trusted subnet - should deny access",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "10.0.0.1",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied\n",
		},
		{
			name:           "IP in trusted subnet - should allow access",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			expectedBody:   `"urls":1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем маршрутизатор с middleware
			r := chi.NewRouter()
			r.Route("/api/internal", func(r chi.Router) {
				r.Use(middleware.TrustedSubnetMiddleware(tt.trustedSubnet, logger))
				r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
					appInstance.HandleStats(w, r)
				})
			})

			// Создаем запрос
			req := createTestRequest(http.MethodGet, "/api/internal/stats", "", nil)
			if tt.clientIP != "" {
				req.Header.Set("X-Real-IP", tt.clientIP)
			}
			rr := httptest.NewRecorder()

			// Вызываем маршрутизатор
			r.ServeHTTP(rr, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}
		})
	}
}
