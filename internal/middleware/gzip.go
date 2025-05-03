package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipMiddleware обрабатывает Gzip-сжатие для запросов и ответов
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Обработка сжатого запроса
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = io.NopCloser(gz)
		}

		// Проверка, поддерживает ли клиент сжатие ответа
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Создаём кастомный ResponseWriter для сжатия ответа
		gw := &gzipResponseWriter{ResponseWriter: w}
		defer gw.Close()

		// Передаём управление следующему обработчику
		next.ServeHTTP(gw, r)
	})
}

// gzipResponseWriter оборачивает http.ResponseWriter для сжатия ответа
type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	isGzipValid bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	// Вызываем оригинальный WriteHeader
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	// Проверяем Content-Type ответа
	contentType := w.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "text/html") {
		w.isGzipValid = false
		return w.ResponseWriter.Write(b)
	}

	// Проверяем размер данных
	if len(b) < 1400 {
		w.isGzipValid = false
		return w.ResponseWriter.Write(b)
	}

	// Инициализируем gzip.Writer, если ещё не создан
	if w.gz == nil {
		w.gz = gzip.NewWriter(w.ResponseWriter)
		w.isGzipValid = true
		w.Header().Set("Content-Encoding", "gzip")
	}

	// Пишем сжатые данные
	n, err := w.gz.Write(b)
	if err != nil {
		return n, err
	}
	return n, nil
}

// Close закрывает gzip.Writer
func (w *gzipResponseWriter) Close() error {
	if w.gz != nil && w.isGzipValid {
		if err := w.gz.Close(); err != nil {
			return err
		}
	}
	return nil
}
