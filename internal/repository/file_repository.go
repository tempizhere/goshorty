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
	UserID      string `json:"user_id,omitempty"`
	DeletedFlag bool   `json:"is_deleted"`
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
			newFile, createErr := os.Create(filePath)
			if createErr != nil {
				return nil, createErr
			}
			if closeErr := newFile.Close(); closeErr != nil {
				return nil, closeErr
			}
			return repo, nil
		}
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			repo.logger.Error("Failed to close file", zap.Error(closeErr))
		}
	}()

	// Читаем файл построчно
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record URLRecord
		if unmarshalErr := json.Unmarshal(scanner.Bytes(), &record); unmarshalErr != nil {
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
		DeletedFlag: false,
	}
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	data = append(data, '\n')

	// Проверяем, существует ли файл, и пытаемся изменить права
	if _, statErr := os.Stat(r.filePath); statErr == nil {
		if chmodErr := os.Chmod(r.filePath, 0644); chmodErr != nil {
			if removeErr := os.Remove(r.filePath); removeErr != nil {
				return "", removeErr
			}
		}
	}

	// Дописываем в файл
	file, err := os.OpenFile(r.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			r.logger.Error("Failed to close file", zap.Error(closeErr))
		}
	}()

	if _, err = file.Write(data); err != nil {
		return "", err
	}
	return id, nil
}

// Get возвращает URL по ID, если он существует
func (r *FileRepository) Get(id string) (models.URL, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	url, exists := r.store[id]
	if !exists {
		return models.URL{}, false
	}

	// Читаем файл для получения UserID и DeletedFlag
	file, err := os.Open(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open file", zap.Error(err))
		return models.URL{}, false
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.logger.Error("Failed to close file", zap.Error(err))
		}
	}()

	var record URLRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if unmarshalErr := json.Unmarshal(scanner.Bytes(), &record); unmarshalErr != nil {
			continue
		}
		if record.ShortURL == id {
			return models.URL{
				ShortID:     id,
				OriginalURL: url,
				UserID:      record.UserID,
				DeletedFlag: record.DeletedFlag,
			}, true
		}
	}
	return models.URL{}, false
}

// Clear очищает хранилище и файл
func (r *FileRepository) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.store = make(map[string]string)
	r.urlToShortID = make(map[string]string)
	if err := os.Remove(r.filePath); err != nil {
		r.logger.Error("Failed to remove file", zap.Error(err))
	}
	newFile, err := os.Create(r.filePath)
	if err == nil {
		if err := newFile.Close(); err != nil {
			r.logger.Error("Failed to close file", zap.Error(err))
		}
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
	defer func() {
		if err := file.Close(); err != nil {
			r.logger.Error("Failed to close file", zap.Error(err))
		}
	}()

	for id, url := range urls {
		record := URLRecord{
			UUID:        id,
			ShortURL:    id,
			OriginalURL: url,
			UserID:      userID,
			DeletedFlag: false,
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

// GetURLsByUserID возвращает все URL, связанные с пользователем
func (r *FileRepository) GetURLsByUserID(userID string) ([]models.URL, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var urls []models.URL
	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return urls, nil
		}
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.logger.Error("Ошибка при закрытии файла", zap.Error(err))
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record URLRecord
		if unmarshalErr := json.Unmarshal(scanner.Bytes(), &record); unmarshalErr != nil {
			r.logger.Warn("Skipping invalid JSON line", zap.String("line", string(scanner.Bytes())), zap.Error(unmarshalErr))
			continue
		}
		if record.UserID == userID {
			urls = append(urls, models.URL{
				ShortID:     record.ShortURL,
				OriginalURL: record.OriginalURL,
				UserID:      record.UserID,
				DeletedFlag: record.DeletedFlag,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

// BatchDelete помечает указанные URL как удалённые
func (r *FileRepository) BatchDelete(userID string, ids []string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Читаем существующие записи
	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			r.logger.Error("Failed to close file", zap.Error(closeErr))
		}
	}()

	var records []URLRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record URLRecord
		if unmarshalErr := json.Unmarshal(scanner.Bytes(), &record); unmarshalErr != nil {
			r.logger.Warn("Skipping invalid JSON line", zap.String("line", string(scanner.Bytes())), zap.Error(unmarshalErr))
			continue
		}
		// Помечаем как удалённые только подходящие записи
		for _, id := range ids {
			if record.ShortURL == id && record.UserID == userID {
				record.DeletedFlag = true
				r.logger.Info("Marked URL as deleted", zap.String("short_id", id), zap.String("user_id", userID))
			}
		}
		records = append(records, record)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return scanErr
	}

	// Переписываем файл
	tmpFile, err := os.CreateTemp(filepath.Dir(r.filePath), "temp_*.json")
	if err != nil {
		return err
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			r.logger.Error("Failed to close temporary file", zap.Error(err))
		}
	}()

	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if _, err := tmpFile.Write(data); err != nil {
			return err
		}
	}

	// Заменяем исходный файл
	if err := os.Rename(tmpFile.Name(), r.filePath); err != nil {
		return err
	}

	return nil
}

// GetStats возвращает статистику сервиса: количество URL и пользователей
func (r *FileRepository) GetStats() (int, int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			r.logger.Error("Failed to close file", zap.Error(closeErr))
		}
	}()

	urlCount := 0
	userSet := make(map[string]struct{})

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record URLRecord
		if unmarshalErr := json.Unmarshal(scanner.Bytes(), &record); unmarshalErr != nil {
			r.logger.Warn("Skipping invalid JSON line", zap.String("line", string(scanner.Bytes())), zap.Error(unmarshalErr))
			continue
		}
		if !record.DeletedFlag {
			urlCount++
			if record.UserID != "" {
				userSet[record.UserID] = struct{}{}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return 0, 0, scanErr
	}

	return urlCount, len(userSet), nil
}

// Close закрывает ресурсы репозитория (убеждается, что все данные записаны в файл)
func (r *FileRepository) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// FileRepository сохраняет данные при каждой операции,
	// поэтому здесь просто логируем закрытие
	r.logger.Info("FileRepository closed", zap.String("file_path", r.filePath))
	return nil
}
