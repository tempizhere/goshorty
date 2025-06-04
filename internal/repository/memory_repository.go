package repository

import "sync"

// MemoryRepository реализует интерфейс Repository с использованием map
type MemoryRepository struct {
	store map[string]string
	mutex sync.RWMutex
}

// NewMemoryRepository создаёт новый экземпляр MemoryRepository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store: make(map[string]string),
		mutex: sync.RWMutex{},
	}
}

// Save сохраняет пару ID-URL в хранилище
func (r *MemoryRepository) Save(id, url string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Проверяем, существует ли original_url
	for shortID, originalURL := range r.store {
		if originalURL == url {
			return shortID, ErrURLExists
		}
	}

	r.store[id] = url
	return id, nil
}

// Get возвращает URL по ID, если он существует
func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	url, exists := r.store[id]
	return url, exists
}

// Clear очищает хранилище
func (r *MemoryRepository) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.store = make(map[string]string)
}

// BatchSave сохраняет множество пар ID-URL в хранилище
func (r *MemoryRepository) BatchSave(urls map[string]string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for id, url := range urls {
		r.store[id] = url
	}
	return nil
}
