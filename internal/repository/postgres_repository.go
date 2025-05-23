package repository

import (
	"database/sql"

	"go.uber.org/zap"
)

// PostgresRepository реализует интерфейс Repository с использованием PostgreSQL
type PostgresRepository struct {
	db     Database
	logger *zap.Logger
}

// NewPostgresRepository создаёт новый экземпляр PostgresRepository
func NewPostgresRepository(db Database, logger *zap.Logger) (*PostgresRepository, error) {
	if db == nil {
		return nil, nil
	}
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}, nil
}

// Save сохраняет пару ID-URL в базе данных
func (r *PostgresRepository) Save(id, url string) error {
	_, err := r.db.Exec("INSERT INTO urls (short_id, original_url) VALUES ($1, $2)", id, url)
	if err != nil {
		r.logger.Error("Failed to save URL to database", zap.String("short_id", id), zap.String("url", url), zap.Error(err))
		return err
	}
	return nil
}

// Get возвращает URL по ID, если он существует
func (r *PostgresRepository) Get(id string) (string, bool) {
	var url string
	err := r.db.QueryRow("SELECT original_url FROM urls WHERE short_id = $1", id).Scan(&url)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		r.logger.Error("Failed to get URL from database", zap.String("short_id", id), zap.Error(err))
		return "", false
	}
	return url, true
}

// Clear очищает все записи в таблице urls
func (r *PostgresRepository) Clear() {
	_, err := r.db.Exec("TRUNCATE TABLE urls RESTART IDENTITY")
	if err != nil {
		r.logger.Error("Failed to clear database", zap.Error(err))
	}
}

// BatchSave сохраняет множество пар ID-URL в базе данных
func (r *PostgresRepository) BatchSave(urls map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		r.logger.Error("Failed to start transaction", zap.Error(err))
		return err
	}
	for id, url := range urls {
		_, err := tx.Exec("INSERT INTO urls (short_id, original_url) VALUES ($1, $2)", id, url)
		if err != nil {
			r.logger.Error("Failed to save URL in transaction", zap.String("short_id", id), zap.String("url", url), zap.Error(err))
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}
	return nil
}
