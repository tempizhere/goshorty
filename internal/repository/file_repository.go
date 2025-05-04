package repository

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
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
}

// NewFileRepository создаёт новый экземпляр FileRepository
func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		store:    make(map[string]string),
		filePath: filePath,
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
			// Пропускаем некорректные строки
			continue
		}
		repo.store[record.ShortURL] = record.OriginalURL
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repo, nil
}

// Save сохраняет пару ID-URL в хранилище и файл
func (r *FileRepository) Save(id, url string) error {
	r.store[id] = url

	// Создаём запись для файла
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

	// Проверяем, существует ли файл, и пытаемся изменить права
	if _, err := os.Stat(r.filePath); err == nil {
		if err := os.Chmod(r.filePath, 0644); err != nil {
			// Если не удалось изменить права, попробуем удалить и пересоздать файл
			if err := os.Remove(r.filePath); err != nil {
				return err
			}
		}
	}

	// Дописываем в файл
	file, err := os.OpenFile(r.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// Get возвращает URL по ID, если он существует
func (r *FileRepository) Get(id string) (string, bool) {
	url, exists := r.store[id]
	return url, exists
}

// Clear очищает хранилище и файл
func (r *FileRepository) Clear() {
	r.store = make(map[string]string)
	// Пересоздаём пустой файл
	os.Remove(r.filePath)
	newFile, err := os.Create(r.filePath)
	if err == nil {
		newFile.Close()
	}
}
