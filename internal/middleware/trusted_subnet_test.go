package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var testLogger *zap.Logger

func init() {
	testLogger, _ = zap.NewDevelopment()
}

func TestTrustedSubnetMiddleware(t *testing.T) {

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
			name:           "Invalid IP address - should deny access",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "invalid-ip",
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
			expectedBody:   "OK",
		},
		{
			name:           "IP at subnet boundary - should allow access",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "192.168.1.1",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "IP at subnet boundary end - should allow access",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "192.168.1.254",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Single IP trusted subnet - should allow access",
			trustedSubnet:  "192.168.1.100/32",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Single IP trusted subnet - should deny access",
			trustedSubnet:  "192.168.1.100/32",
			clientIP:       "192.168.1.101",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied\n",
		},
		{
			name:           "Large subnet - should allow access",
			trustedSubnet:  "10.0.0.0/8",
			clientIP:       "10.255.255.255",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем middleware
			middleware := TrustedSubnetMiddleware(tt.trustedSubnet, testLogger)

			// Создаем тестовый обработчик
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("OK")); err != nil {
					t.Logf("Failed to write response: %v", err)
				}
			})

			// Создаем запрос
			req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
			if tt.clientIP != "" {
				req.Header.Set("X-Real-IP", tt.clientIP)
			}

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Вызываем middleware
			middleware(handler).ServeHTTP(rr, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestTrustedSubnetMiddleware_InvalidCIDR(t *testing.T) {
	// Создаем middleware с невалидной CIDR-нотацией
	middleware := TrustedSubnetMiddleware("invalid-cidr", testLogger)

	// Создаем тестовый обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	})

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Вызываем middleware
	middleware(handler).ServeHTTP(rr, req)

	// Проверяем, что возвращается ошибка сервера
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "Internal server error\n", rr.Body.String())
}

func TestTrustedSubnetMiddleware_IPv6(t *testing.T) {

	tests := []struct {
		name           string
		trustedSubnet  string
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "IPv6 in trusted subnet - should allow access",
			trustedSubnet:  "2001:db8::/32",
			clientIP:       "2001:db8::1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IPv6 not in trusted subnet - should deny access",
			trustedSubnet:  "2001:db8::/32",
			clientIP:       "2001:db9::1",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем middleware
			middleware := TrustedSubnetMiddleware(tt.trustedSubnet, testLogger)

			// Создаем тестовый обработчик
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("OK")); err != nil {
					t.Logf("Failed to write response: %v", err)
				}
			})

			// Создаем запрос
			req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
			req.Header.Set("X-Real-IP", tt.clientIP)

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Вызываем middleware
			middleware(handler).ServeHTTP(rr, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
