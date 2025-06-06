package repository

import (
	"sync"

	"github.com/tempizhere/goshorty/internal/models"
)

// MemoryRepository реализует интерфейс Repository с использованием map
type MemoryRepository struct {
	store map[string]models.URL
	mutex sync.RWMutex
}

// NewMemoryRepository создаёт новый экземпляр MemoryRepository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store: make(map[string]models.URL),
		mutex: sync.RWMutex{},
	}
}

// Save сохраняет пару ID-URL в хранилище
func (r *MemoryRepository) Save(id, url, userID string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Проверяем, существует ли original_url
	for shortID, u := range r.store {
		if u.OriginalURL == url {
			return shortID, ErrURLExists
		}
	}

	r.store[id] = models.URL{
		ShortID:     id,
		OriginalURL: url,
		UserID:      userID,
	}
	return id, nil
}

// Get возвращает URL по ID, если он существует
func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	u, exists := r.store[id]
	return u.OriginalURL, exists
}

// Clear очищает хранилище
func (r *MemoryRepository) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.store = make(map[string]models.URL)
}

// BatchSave сохраняет множество пар ID-URL в хранилище
func (r *MemoryRepository) BatchSave(urls map[string]string, userID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for id, url := range urls {
		for _, u := range r.store {
			if u.OriginalURL == url {
				return ErrURLExists
			}
		}
		r.store[id] = models.URL{
			ShortID:     id,
			OriginalURL: url,
			UserID:      userID,
		}
	}
	return nil
}

// GetURLsByUserID возвращает все URL, связанные с пользователем
func (r *MemoryRepository) GetURLsByUserID(userID string) ([]models.URL, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var urls []models.URL
	for _, u := range r.store {
		if u.UserID == userID {
			urls = append(urls, u)
		}
	}
	return urls, nil
}
