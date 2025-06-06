package repository

import (
	"database/sql"
	"errors"

	"github.com/tempizhere/goshorty/internal/models"
)

var (
	ErrURLExists = errors.New("URL already exists")
)

type Repository interface {
	Save(id, url, userID string) (string, error)
	Get(id string) (string, bool)
	Clear()
	BatchSave(urls map[string]string, userID string) error
	GetURLsByUserID(userID string) ([]models.URL, error)
}

type Database interface {
	Ping() error
	Close() error
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
}
