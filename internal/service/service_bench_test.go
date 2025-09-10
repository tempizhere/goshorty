package service

import (
	"fmt"
	"testing"

	"github.com/tempizhere/goshorty/internal/models"
)

// BenchmarkRepository для бенчмарков
type benchmarkRepository struct {
	urls map[string]models.URL
}

func newBenchmarkRepository() *benchmarkRepository {
	return &benchmarkRepository{
		urls: make(map[string]models.URL),
	}
}

func (m *benchmarkRepository) Save(id, url, userID string) (string, error) {
	m.urls[id] = models.URL{
		ShortID:     id,
		OriginalURL: url,
		UserID:      userID,
	}
	return id, nil
}

func (m *benchmarkRepository) Get(id string) (models.URL, bool) {
	url, exists := m.urls[id]
	return url, exists
}

func (m *benchmarkRepository) Clear() {
	m.urls = make(map[string]models.URL)
}

func (m *benchmarkRepository) BatchSave(urls map[string]string, userID string) error {
	for id, url := range urls {
		m.urls[id] = models.URL{
			ShortID:     id,
			OriginalURL: url,
			UserID:      userID,
		}
	}
	return nil
}

func (m *benchmarkRepository) GetURLsByUserID(userID string) ([]models.URL, error) {
	var result []models.URL
	for _, url := range m.urls {
		if url.UserID == userID {
			result = append(result, url)
		}
	}
	return result, nil
}

func (m *benchmarkRepository) BatchDelete(userID string, ids []string) error {
	return nil
}

func (m *benchmarkRepository) GetStats() (int, int, error) {
	urlCount := 0
	userSet := make(map[string]struct{})

	for _, u := range m.urls {
		if !u.DeletedFlag {
			urlCount++
			if u.UserID != "" {
				userSet[u.UserID] = struct{}{}
			}
		}
	}

	return urlCount, len(userSet), nil
}

func (m *benchmarkRepository) Close() error {
	// Benchmark repository не имеет ресурсов для закрытия
	return nil
}

// Бенчмарки для генерации коротких ID
func BenchmarkGenerateShortID(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.GenerateShortID()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Бенчмарки для создания коротких URL
func BenchmarkCreateShortURL(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.CreateShortURL("https://example.com/very/long/url/that/needs/to/be/shortened", "user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Бенчмарки для создания коротких URL с заданным ID
func BenchmarkCreateShortURLWithID(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Используем уникальный ID для каждой итерации
		id := fmt.Sprintf("test%d", i)
		_, err := svc.CreateShortURLWithID("https://example.com/very/long/url/that/needs/to/be/shortened", id, "user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Бенчмарки для получения оригинального URL
func BenchmarkGetOriginalURL(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	// Подготавливаем данные
	_, err := svc.CreateShortURLWithID("https://example.com/very/long/url/that/needs/to/be/shortened", "test123", "user123")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, exists := svc.GetOriginalURL("test123")
		if !exists {
			b.Fatal("URL not found")
		}
	}
}

// Бенчмарки для генерации JWT
func BenchmarkGenerateJWT(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.GenerateJWT("user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Бенчмарки для парсинга JWT
func BenchmarkParseJWT(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	// Подготавливаем токен
	token, err := svc.GenerateJWT("user123")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.ParseJWT(token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Бенчмарки для пакетного сокращения URL
func BenchmarkBatchShorten(b *testing.B) {
	svc := NewService(newBenchmarkRepository(), "http://localhost:8080", "secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Подготавливаем данные с уникальными correlation_id
		reqs := []models.BatchRequest{
			{CorrelationID: fmt.Sprintf("1_%d", i), OriginalURL: "https://example1.com/very/long/url"},
			{CorrelationID: fmt.Sprintf("2_%d", i), OriginalURL: "https://example2.com/very/long/url"},
			{CorrelationID: fmt.Sprintf("3_%d", i), OriginalURL: "https://example3.com/very/long/url"},
			{CorrelationID: fmt.Sprintf("4_%d", i), OriginalURL: "https://example4.com/very/long/url"},
			{CorrelationID: fmt.Sprintf("5_%d", i), OriginalURL: "https://example5.com/very/long/url"},
		}

		_, err := svc.BatchShorten(reqs, "user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}
