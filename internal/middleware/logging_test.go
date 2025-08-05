package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggingMiddleware(t *testing.T) {
	logger := zap.NewNop()
	middleware := LoggingMiddleware(logger)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test response")); err != nil {
			t.Logf("Ошибка при записи в response: %v", err)
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
}

func TestLoggingMiddleware_DifferentMethods(t *testing.T) {
	logger := zap.NewNop()
	middleware := LoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("response")); err != nil {
			t.Logf("Ошибка при записи в response: %v", err)
		}
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			middleware(handler).ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "response", w.Body.String())
		})
	}
}

func TestLoggingMiddleware_DifferentStatusCodes(t *testing.T) {
	logger := zap.NewNop()
	middleware := LoggingMiddleware(logger)

	statusCodes := []int{200, 201, 400, 404, 500}

	for _, statusCode := range statusCodes {
		t.Run("Status"+string(rune(statusCode)), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
				if _, err := w.Write([]byte("response")); err != nil {
					t.Logf("Ошибка при записи в response: %v", err)
				}
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			middleware(handler).ServeHTTP(w, req)

			assert.Equal(t, statusCode, w.Code)
			assert.Equal(t, "response", w.Body.String())
		})
	}
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()

	lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	assert.Equal(t, http.StatusOK, lw.statusCode)
	assert.Equal(t, 0, lw.size)

	lw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, lw.statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()

	lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	assert.Equal(t, 0, lw.size)

	data := []byte("test data")
	n, err := lw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), lw.size)
	assert.Equal(t, string(data), w.Body.String())

	moreData := []byte(" more")
	n, err = lw.Write(moreData)

	assert.NoError(t, err)
	assert.Equal(t, len(moreData), n)
	assert.Equal(t, len(data)+len(moreData), lw.size)
	assert.Equal(t, string(data)+string(moreData), w.Body.String())
}
