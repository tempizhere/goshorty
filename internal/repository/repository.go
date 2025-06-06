package repository

import (
	"database/sql"
	"errors"
)

var (
	ErrURLExists = errors.New("URL already exists")
)

type Repository interface {
	Save(id, url string) (string, error)
	Get(id string) (string, bool)
	Clear()
	BatchSave(urls map[string]string) error
}

type Database interface {
	Ping() error
	Close() error
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
}
