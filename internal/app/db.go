package app

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database interface {
	Ping() error
	Close() error
}

type DB struct {
	conn *sql.DB
}

func NewDB(dsn string) (*DB, error) {
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

func (db *DB) Ping() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Ping()
}

func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Close()
}
