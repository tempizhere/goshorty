package repository

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

// URLRecord представляет запись в JSON-файле
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// FileRepository реализует интерфейс Repository с использованием файла
type FileRepository struct {
	store    map[string]string
	filePath string
	logger   *zap.Logger
	mutex    sync.RWMutex
}

// NewFileRepository создаёт новый экземпляр FileRepository
func NewFileRepository(filePath string, logger *zap.Logger) (*FileRepository, error) {
	repo := &FileRepository{
		store:    make(map[string]string),
		filePath: filePath,
		logger:   logger,
	}

	// Создаём директорию, если не существует
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Читаем существующий файл, если он есть
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создадим пустой
			newFile, err := os.Create(filePath)
			if err != nil {
				return nil, err
			}
			newFile.Close()
			return repo, nil
		}
		return nil, err
	}
	defer file.Close()

	// Читаем файл построчно
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record URLRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			// Пропускаем некорректные строки и логируем это
			repo.logger.Warn("Skipping invalid JSON line", zap.String("line", string(scanner.Bytes())), zap.Error(err))
			continue
		}
		repo.mutex.Lock()
		repo.store[record.ShortURL] = record.OriginalURL
		repo.mutex.Unlock()
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repo, nil
}

// Save сохраняет пару ID-URL в хранилище и файл
func (r *FileRepository) Save(id, url string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Проверяем, существует ли original_url
	for shortID, originalURL := range r.store {
		if originalURL == url {
			r.logger.Info("URL already exists", zap.String("original_url", url), zap.String("short_id", shortID))
			return shortID, ErrURLExists
		}
	}

	r.store[id] = url

	// Создаём запись для файла
	record := URLRecord{
		UUID:        id,
		ShortURL:    id,
		OriginalURL: url,
	}
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	data = append(data, '\n')

	// Проверяем, существует ли файл, и пытаемся изменить права
	if _, err := os.Stat(r.filePath); err == nil {
		if err := os.Chmod(r.filePath, 0644); err != nil {
			// Если не удалось изменить права, попробуем удалить и пересоздать файл
			if err := os.Remove(r.filePath); err != nil {
				return "", err
			}
		}
	}

	// Дописываем в файл
	file, err := os.OpenFile(r.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err = file.Write(data); err != nil {
		return "", err
	}
	return id, nil
}

// Get возвращает URL по ID, если он существует
func (r *FileRepository) Get(id string) (string, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	url, exists := r.store[id]
	return url, exists
}

// Clear очищает хранилище и файл
func (r *FileRepository) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.store = make(map[string]string)
	// Пересоздаём пустой файл
	os.Remove(r.filePath)
	newFile, err := os.Create(r.filePath)
	if err == nil {
		newFile.Close()
	}
}

// BatchSave сохраняет множество пар ID-URL в хранилище и файл
func (r *FileRepository) BatchSave(urls map[string]string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for id, url := range urls {
		r.store[id] = url
	}

	file, err := os.OpenFile(r.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for id, url := range urls {
		record := URLRecord{
			UUID:        id,
			ShortURL:    id,
			OriginalURL: url,
		}
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if _, err := file.Write(data); err != nil {
			return err
		}
	}
	return nil
}
