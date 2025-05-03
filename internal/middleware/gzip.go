package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipMiddleware обрабатывает Gzip-сжатие для запросов
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Обработка сжатого запроса
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = io.NopCloser(gz)
		}

		// Передаём запрос дальше без сжатия ответа
		next.ServeHTTP(w, r)
	})
}
