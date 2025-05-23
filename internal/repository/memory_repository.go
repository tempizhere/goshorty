package repository

// MemoryRepository реализует интерфейс Repository с использованием map
type MemoryRepository struct {
	store map[string]string
}

// NewMemoryRepository создаёт новый экземпляр MemoryRepository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store: make(map[string]string),
	}
}

// Save сохраняет пару ID-URL в хранилище
func (r *MemoryRepository) Save(id, url string) error {
	r.store[id] = url
	return nil
}

// Get возвращает URL по ID, если он существует
func (r *MemoryRepository) Get(id string) (string, bool) {
	url, exists := r.store[id]
	return url, exists
}

// Clear очищает хранилище
func (r *MemoryRepository) Clear() {
	r.store = make(map[string]string)
}

// BatchSave сохраняет множество пар ID-URL в хранилище
func (r *MemoryRepository) BatchSave(urls map[string]string) error {
	for id, url := range urls {
		r.store[id] = url
	}
	return nil
}
