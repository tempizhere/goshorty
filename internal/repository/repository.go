package repository

import (
	"database/sql"
	"errors"

	"github.com/tempizhere/goshorty/internal/models"
)

// ErrURLExists возвращается при попытке сохранить URL, который уже существует
var ErrURLExists = errors.New("URL already exists")

// Repository определяет интерфейс для работы с хранилищем URL
type Repository interface {
	// Save сохраняет URL с заданным ID и возвращает короткий ID или ошибку
	Save(id, url, userID string) (string, error)
	// Get возвращает URL по короткому ID и флаг существования
	Get(id string) (models.URL, bool)
	// Clear очищает все данные в хранилище
	Clear()
	// BatchSave сохраняет несколько URL для одного пользователя
	BatchSave(urls map[string]string, userID string) error
	// GetURLsByUserID возвращает все URL, созданные пользователем
	GetURLsByUserID(userID string) ([]models.URL, error)
	// BatchDelete помечает URL как удалённые для указанного пользователя
	BatchDelete(userID string, ids []string) error
}

// Database определяет интерфейс для работы с базой данных
type Database interface {
	// Ping проверяет соединение с базой данных
	Ping() error
	// Close закрывает соединение с базой данных
	Close() error
	// Exec выполняет SQL-команду без возврата результатов
	Exec(query string, args ...interface{}) (sql.Result, error)
	// Query выполняет SQL-запрос и возвращает результаты
	Query(query string, args ...interface{}) (*sql.Rows, error)
	// QueryRow выполняет SQL-запрос и возвращает одну строку результата
	QueryRow(query string, args ...interface{}) *sql.Row
	// Begin начинает новую транзакцию
	Begin() (*sql.Tx, error)
}
