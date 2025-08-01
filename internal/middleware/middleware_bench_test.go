package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

var benchLogger *zap.Logger

func init() {
	benchLogger, _ = zap.NewDevelopment()
}

// BenchmarkLoggingMiddleware измеряет производительность middleware логирования
func BenchmarkLoggingMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggingMiddleware := LoggingMiddleware(benchLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		w := httptest.NewRecorder()
		loggingMiddleware(handler).ServeHTTP(w, req)
	}
}

// BenchmarkGzipMiddleware измеряет производительность middleware сжатия
func BenchmarkGzipMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test response data"))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		w := httptest.NewRecorder()
		GzipMiddleware(handler).ServeHTTP(w, req)
	}
}

// BenchmarkGzipMiddlewareWithoutCompression измеряет производительность middleware сжатия без сжатия
func BenchmarkGzipMiddlewareWithoutCompression(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test response data"))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		w := httptest.NewRecorder()
		GzipMiddleware(handler).ServeHTTP(w, req)
	}
}

// BenchmarkConcurrentLoggingMiddleware измеряет производительность конкурентного middleware логирования
func BenchmarkConcurrentLoggingMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	loggingMiddleware := LoggingMiddleware(benchLogger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			w := httptest.NewRecorder()
			loggingMiddleware(handler).ServeHTTP(w, req)
		}
	})
}

// BenchmarkConcurrentGzipMiddleware измеряет производительность конкурентного middleware сжатия
func BenchmarkConcurrentGzipMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test response data"))
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			w := httptest.NewRecorder()
			GzipMiddleware(handler).ServeHTTP(w, req)
		}
	})
}

// BenchmarkLargeResponseGzipMiddleware измеряет производительность middleware сжатия с большим ответом
func BenchmarkLargeResponseGzipMiddleware(b *testing.B) {
	largeResponse := make([]byte, 10000)
	for i := range largeResponse {
		largeResponse[i] = byte(i % 256)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(largeResponse)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		w := httptest.NewRecorder()
		GzipMiddleware(handler).ServeHTTP(w, req)
	}
}
