package app

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tempizhere/goshorty/internal/repository"
)

// DB представляет подключение к базе данных
type DB struct {
	conn *sql.DB
}

// NewDB создаёт новое подключение к базе данных
func NewDB(dsn string) (repository.Database, error) {
	if dsn == "" {
		return nil, nil
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
	}

	if dsn != "" {
		_, err := conn.Exec(`
            CREATE TABLE IF NOT EXISTS urls (
                id SERIAL PRIMARY KEY,
                short_id VARCHAR(10) UNIQUE NOT NULL,
                original_url TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        `)
		if err != nil {
			conn.Close()
			return nil, err
		}
	}

	return &DB{conn: conn}, nil
}

// Ping проверяет соединение с базой данных
func (db *DB) Ping() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Ping()
}

// Close закрывает соединение с базой данных
func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

// Exec выполняет SQL-запрос с аргументами
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// QueryRow выполняет SQL-запрос и возвращает одну строку
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// Begin начинает транзакцию
func (db *DB) Begin() (*sql.Tx, error) {
	if db == nil || db.conn == nil {
		return nil, sql.ErrConnDone
	}
	return db.conn.Begin()
}
