package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// loggingResponseWriter оборачивает http.ResponseWriter для отслеживания статуса и размера ответа
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader перехватывает код статуса
func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write перехватывает размер ответа
func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// LoggingMiddleware создаёт middleware для логирования запросов и ответов
func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Оборачиваем ResponseWriter
			lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Вызываем следующий обработчик
			next.ServeHTTP(lw, r)

			// Логируем запрос и ответ
			duration := time.Since(start)
			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.Int("status", lw.statusCode),
				zap.Int("size", lw.size),
				zap.Duration("duration_ms", duration/time.Millisecond),
			)
		})
	}
}
