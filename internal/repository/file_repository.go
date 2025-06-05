package repository

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/tempizhere/goshorty/internal/models"
	"go.uber.org/zap"
)

// URLRecord представляет запись в JSON-файле
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

// FileRepository реализует интерфейс Repository с использованием файла
type FileRepository struct {
	store        map[string]string // short_id -> original_url
	urlToShortID map[string]string // original_url -> short_id
	filePath     string
	logger       *zap.Logger
	mutex        sync.RWMutex
}

// NewFileRepository создаёт новый экземпляр FileRepository
func NewFileRepository(filePath string, logger *zap.Logger) (*FileRepository, error) {
	repo := &FileRepository{
		store:        make(map[string]string),
		urlToShortID: make(map[string]string),
		filePath:     filePath,
		logger:       logger,
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
		repo.urlToShortID[record.OriginalURL] = record.ShortURL
		repo.mutex.Unlock()
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repo, nil
}

// Save сохраняет пару ID-URL в хранилище и файл
func (r *FileRepository) Save(id, url, userID string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Проверяем, существует ли original_url
	if shortID, exists := r.urlToShortID[url]; exists {
		r.logger.Info("URL already exists", zap.String("original_url", url), zap.String("short_id", shortID))
		return shortID, ErrURLExists
	}

	r.store[id] = url
	r.urlToShortID[url] = id

	// Создаём запись для файла
	record := URLRecord{
		UUID:        id,
		ShortURL:    id,
		OriginalURL: url,
		UserID:      userID,
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
	r.urlToShortID = make(map[string]string)
	// Пересоздаём пустой файл
	os.Remove(r.filePath)
	newFile, err := os.Create(r.filePath)
	if err == nil {
		newFile.Close()
	}
}

// BatchSave сохраняет множество пар ID-URL в хранилище и файл
func (r *FileRepository) BatchSave(urls map[string]string, userID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for id, url := range urls {
		if shortID, exists := r.urlToShortID[url]; exists {
			r.logger.Info("URL already exists in batch", zap.String("original_url", url), zap.String("short_id", shortID))
			return ErrURLExists
		}
		r.store[id] = url
		r.urlToShortID[url] = id
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
			UserID:      userID,
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

// GetURLsByUserID возвращает все URL, созданные пользователем
func (r *FileRepository) GetURLsByUserID(userID string) ([]models.URL, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var urls []models.URL

	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return urls, nil // Файл не существует, возвращаем пустой список
		}
		r.logger.Error("Failed to open file", zap.Error(err))
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record URLRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			r.logger.Warn("Skipping invalid JSON line", zap.String("line", string(scanner.Bytes())), zap.Error(err))
			continue
		}
		if record.UserID == userID {
			urls = append(urls, models.URL{
				ShortID:     record.ShortURL,
				OriginalURL: record.OriginalURL,
				UserID:      record.UserID,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		r.logger.Error("Error reading file", zap.Error(err))
		return nil, err
	}

	return urls, nil
}
