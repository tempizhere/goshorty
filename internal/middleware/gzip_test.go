package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipMiddleware_NoCompression(t *testing.T) {
	middleware := GzipMiddleware

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test response"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	assert.Equal(t, "", w.Header().Get("Content-Encoding"))
}

func TestGzipMiddleware_WithCompression(t *testing.T) {
	middleware := GzipMiddleware

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Header().Set("Content-Type", "application/json")
		largeResponse := strings.Repeat("test data ", 200) // ~2000 байт
		w.Write([]byte(largeResponse))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

	body := w.Body.Bytes()
	assert.NotEqual(t, "test data test data test data ", string(body[:30]))
}

func TestGzipMiddleware_SmallResponse(t *testing.T) {
	middleware := GzipMiddleware

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("small"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "small", w.Body.String())
}

func TestGzipMiddleware_UnsupportedContentType(t *testing.T) {
	middleware := GzipMiddleware

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Header().Set("Content-Type", "image/png")
		largeResponse := strings.Repeat("test data ", 200)
		w.Write([]byte(largeResponse))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "", w.Header().Get("Content-Encoding"))
}

func TestGzipMiddleware_GzipRequest(t *testing.T) {
	middleware := GzipMiddleware

	var buf strings.Builder
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("compressed data"))
	gw.Close()
	compressedData := buf.String()

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "compressed data", string(body))
		w.Write([]byte("response"))
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader(compressedData))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "response", w.Body.String())
}

func TestGzipResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()

	gw := &gzipResponseWriter{ResponseWriter: w}

	gw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGzipResponseWriter_Close(t *testing.T) {
	w := httptest.NewRecorder()

	gw := &gzipResponseWriter{ResponseWriter: w}

	err := gw.Close()
	assert.NoError(t, err)

	gw.gz = gzip.NewWriter(w)
	gw.isGzipValid = true
	err = gw.Close()
	assert.NoError(t, err)
}
