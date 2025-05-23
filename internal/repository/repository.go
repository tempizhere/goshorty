package repository

import "database/sql"

type Repository interface {
	Save(id, url string) error
	Get(id string) (string, bool)
	Clear()
}

type Database interface {
	Ping() error
	Close() error
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
