package repository

import (
	"strconv"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
)

// BenchmarkMemoryRepository_Save измеряет производительность сохранения в memory репозитории
func BenchmarkMemoryRepository_Save(b *testing.B) {
	repo := NewMemoryRepository()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := "test-id-" + strconv.Itoa(i)
		url := "https://example.com/url/" + strconv.Itoa(i)
		_, err := repo.Save(id, url, "test-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_Get измеряет производительность получения из memory репозитория
func BenchmarkMemoryRepository_Get(b *testing.B) {
	repo := NewMemoryRepository()

	// Подготавливаем данные
	id := "test-id"
	url := "https://example.com/test-url"
	if _, err := repo.Save(id, url, "test-user"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, exists := repo.Get(id)
		if !exists {
			b.Fatal("URL not found")
		}
	}
}

// BenchmarkMemoryRepository_BatchSave измеряет производительность пакетного сохранения в memory репозитории
func BenchmarkMemoryRepository_BatchSave(b *testing.B) {
	repo := NewMemoryRepository()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		urls := make(map[string]string)
		for j := 0; j < 10; j++ {
			id := "batch-id-" + strconv.Itoa(i) + "-" + strconv.Itoa(j)
			url := "https://example.com/batch/" + strconv.Itoa(i) + "-" + strconv.Itoa(j)
			urls[id] = url
		}
		err := repo.BatchSave(urls, "test-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_GetURLsByUserID измеряет производительность получения URL пользователя из memory репозитория
func BenchmarkMemoryRepository_GetURLsByUserID(b *testing.B) {
	repo := NewMemoryRepository()

	// Подготавливаем данные - создаем несколько URL для пользователя
	userID := "test-user"
	for i := 0; i < 10; i++ {
		id := "user-id-" + strconv.Itoa(i)
		url := "https://example.com/user/" + strconv.Itoa(i)
		if _, err := repo.Save(id, url, userID); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetURLsByUserID(userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_BatchDelete измеряет производительность пакетного удаления из memory репозитория
func BenchmarkMemoryRepository_BatchDelete(b *testing.B) {
	repo := NewMemoryRepository()

	// Подготавливаем данные
	userID := "test-user"
	var ids []string
	for i := 0; i < 5; i++ {
		id := "delete-id-" + strconv.Itoa(i)
		url := "https://example.com/delete/" + strconv.Itoa(i)
		if _, err := repo.Save(id, url, userID); err != nil {
			b.Fatal(err)
		}
		ids = append(ids, id)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.BatchDelete(userID, ids)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFileRepository_Save измеряет производительность сохранения в file репозитории
func BenchmarkFileRepository_Save(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	repo, err := NewFileRepository("benchmark_test.json", logger)
	if err != nil {
		b.Fatal(err)
	}
	defer repo.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := "file-id-" + strconv.Itoa(i)
		url := "https://example.com/file/" + strconv.Itoa(i)
		_, err := repo.Save(id, url, "test-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFileRepository_Get измеряет производительность получения из file репозитория
func BenchmarkFileRepository_Get(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	repo, err := NewFileRepository("benchmark_test.json", logger)
	if err != nil {
		b.Fatal(err)
	}
	defer repo.Clear()

	// Подготавливаем данные
	id := "file-test-id"
	url := "https://example.com/file-test-url"
	if _, err := repo.Save(id, url, "test-user"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, exists := repo.Get(id)
		if !exists {
			b.Fatal("URL not found")
		}
	}
}

// BenchmarkFileRepository_BatchSave измеряет производительность пакетного сохранения в file репозитории
func BenchmarkFileRepository_BatchSave(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	repo, err := NewFileRepository("benchmark_test.json", logger)
	if err != nil {
		b.Fatal(err)
	}
	defer repo.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		urls := make(map[string]string)
		for j := 0; j < 5; j++ {
			id := "file-batch-id-" + strconv.Itoa(i) + "-" + strconv.Itoa(j)
			url := "https://example.com/file-batch/" + strconv.Itoa(i) + "-" + strconv.Itoa(j)
			urls[id] = url
		}
		err := repo.BatchSave(urls, "test-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentMemoryRepository_Save измеряет производительность конкурентного сохранения в memory репозитории
func BenchmarkConcurrentMemoryRepository_Save(b *testing.B) {
	repo := NewMemoryRepository()
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt64(&counter, 1) - 1
			id := "concurrent-id-" + strconv.FormatInt(i, 10)
			url := "https://example.com/concurrent/" + strconv.FormatInt(i, 10)
			_, err := repo.Save(id, url, "test-user")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentMemoryRepository_Get измеряет производительность конкурентного получения из memory репозитория
func BenchmarkConcurrentMemoryRepository_Get(b *testing.B) {
	repo := NewMemoryRepository()

	// Подготавливаем данные
	for i := 0; i < 100; i++ {
		id := "concurrent-get-id-" + strconv.Itoa(i)
		url := "https://example.com/concurrent-get/" + strconv.Itoa(i)
		if _, err := repo.Save(id, url, "test-user"); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := "concurrent-get-id-" + strconv.Itoa(i%100)
			_, exists := repo.Get(id)
			if !exists {
				b.Fatal("URL not found")
			}
			i++
		}
	})
}

// BenchmarkMemoryRepository_LargeDataset измеряет производительность работы с большим набором данных
func BenchmarkMemoryRepository_LargeDataset(b *testing.B) {
	repo := NewMemoryRepository()

	// Подготавливаем большой набор данных
	for i := 0; i < 1000; i++ {
		id := "large-id-" + strconv.Itoa(i)
		url := "https://example.com/large/" + strconv.Itoa(i)
		if _, err := repo.Save(id, url, "test-user"); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := "large-id-" + strconv.Itoa(i%1000)
		_, exists := repo.Get(id)
		if !exists {
			b.Fatal("URL not found")
		}
	}
}
