package repository

import (
	"database/sql"

	"github.com/jackc/pgx/v5/pgconn"
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
func (r *PostgresRepository) Save(id, url string) (string, error) {
	// Сначала проверяем, существует ли original_url
	var existingID string
	err := r.db.QueryRow("SELECT short_id FROM urls WHERE original_url = $1", url).Scan(&existingID)
	if err == nil {
		r.logger.Info("URL already exists",
			zap.String("original_url", url),
			zap.String("existing_short_id", existingID))
		return existingID, ErrURLExists
	}
	if err != sql.ErrNoRows {
		r.logger.Error("Failed to check existing URL",
			zap.String("original_url", url),
			zap.Error(err))
		return "", err
	}

	// Если URL не существует, выполняем INSERT
	var shortID string
	err = r.db.QueryRow(`
		INSERT INTO urls (short_id, original_url)
		VALUES ($1, $2)
		ON CONFLICT (original_url)
		DO UPDATE SET short_id = urls.short_id
		RETURNING short_id
	`, id, url).Scan(&shortID)
	if err != nil {
		r.logger.Error("Failed to execute INSERT with ON CONFLICT",
			zap.String("short_id", id),
			zap.String("original_url", url),
			zap.Error(err))
		if pgErr, ok := err.(*pgconn.PgError); ok {
			r.logger.Debug("PostgreSQL error details",
				zap.String("code", pgErr.Code),
				zap.String("message", pgErr.Message),
				zap.String("constraint", pgErr.ConstraintName))
		}
		return "", err
	}
	if shortID != id {
		r.logger.Info("URL already exists",
			zap.String("original_url", url),
			zap.String("existing_short_id", shortID))
		return shortID, ErrURLExists
	}
	r.logger.Info("URL saved successfully",
		zap.String("short_id", id),
		zap.String("original_url", url))
	return id, nil
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
		var shortID string
		err := tx.QueryRow(`
			INSERT INTO urls (short_id, original_url)
			VALUES ($1, $2)
			ON CONFLICT (original_url)
			DO UPDATE SET short_id = urls.short_id
			RETURNING short_id
		`, id, url).Scan(&shortID)
		if err != nil {
			r.logger.Error("Failed to save URL in transaction",
				zap.String("short_id", id),
				zap.String("original_url", url),
				zap.Error(err))
			tx.Rollback()
			return err
		}
		if shortID != id {
			r.logger.Info("URL already exists in transaction",
				zap.String("original_url", url),
				zap.String("existing_short_id", shortID))
			tx.Rollback()
			return ErrURLExists
		}
	}
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}
	return nil
}
