package repository

import (
	"database/sql"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/tempizhere/goshorty/internal/models"
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
	repo := &PostgresRepository{
		db:     db,
		logger: logger,
	}

	// Добавляем столбец user_id, если он не существует
	_, err := db.Exec("ALTER TABLE urls ADD COLUMN IF NOT EXISTS user_id VARCHAR")
	if err != nil {
		logger.Error("Failed to add user_id column", zap.Error(err))
		return nil, err
	}

	// Добавляем столбец is_deleted, если он не существует
	_, err = db.Exec("ALTER TABLE urls ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT FALSE")
	if err != nil {
		logger.Error("Failed to add is_deleted column", zap.Error(err))
		return nil, err
	}

	return repo, nil
}

// Save сохраняет пару ID-URL в базе данных
func (r *PostgresRepository) Save(id, url, userID string) (string, error) {
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
	query := `
		INSERT INTO urls (short_id, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (original_url)
		DO UPDATE SET short_id = urls.short_id
		RETURNING short_id
	`
	var userIDValue interface{}
	if userID == "" {
		userIDValue = nil
	} else {
		userIDValue = userID
	}
	err = r.db.QueryRow(query, id, url, userIDValue).Scan(&shortID)
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
func (r *PostgresRepository) Get(id string) (models.URL, bool) {
	var u models.URL
	var userID sql.NullString
	err := r.db.QueryRow("SELECT short_id, original_url, user_id, is_deleted FROM urls WHERE short_id = $1", id).
		Scan(&u.ShortID, &u.OriginalURL, &userID, &u.DeletedFlag)
	if err == sql.ErrNoRows {
		return models.URL{}, false
	}
	if err != nil {
		r.logger.Error("Failed to get URL from database", zap.String("short_id", id), zap.Error(err))
		return models.URL{}, false
	}
	u.UserID = userID.String
	return u, true
}

// Clear очищает все записи в таблице urls
func (r *PostgresRepository) Clear() {
	_, err := r.db.Exec("TRUNCATE TABLE urls RESTART IDENTITY")
	if err != nil {
		r.logger.Error("Failed to clear database", zap.Error(err))
	}
}

// BatchSave сохраняет множество пар ID-URL в базе данных
func (r *PostgresRepository) BatchSave(urls map[string]string, userID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		r.logger.Error("Failed to start transaction", zap.Error(err))
		return err
	}
	for id, url := range urls {
		var shortID string
		query := `
			INSERT INTO urls (short_id, original_url, user_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (original_url)
			DO UPDATE SET short_id = urls.short_id
			RETURNING short_id
		`
		var userIDValue interface{}
		if userID == "" {
			userIDValue = nil
		} else {
			userIDValue = userID
		}
		err := tx.QueryRow(query, id, url, userIDValue).Scan(&shortID)
		if err != nil {
			r.logger.Error("Failed to save URL in transaction",
				zap.String("short_id", id),
				zap.String("original_url", url),
				zap.Error(err))
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			}
			return err
		}
		if shortID != id {
			r.logger.Info("URL already exists in transaction",
				zap.String("original_url", url),
				zap.String("existing_short_id", shortID))
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			}
			return ErrURLExists
		}
	}
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}
	return nil
}

// Close закрывает ресурсы репозитория (соединение с базой данных)
func (r *PostgresRepository) Close() error {
	if r.db != nil {
		r.logger.Info("Closing PostgreSQL repository")
		return r.db.Close()
	}
	return nil
}

// GetURLsByUserID возвращает все URL, связанные с пользователем
func (r *PostgresRepository) GetURLsByUserID(userID string) ([]models.URL, error) {
	rows, err := r.db.Query("SELECT short_id, original_url, user_id, is_deleted FROM urls WHERE user_id = $1 AND is_deleted = FALSE", userID)
	if err != nil {
		r.logger.Error("Failed to query URLs by user_id", zap.String("user_id", userID), zap.Error(err))
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("Failed to close rows", zap.Error(err))
		}
	}()

	var urls []models.URL
	for rows.Next() {
		var u models.URL
		var userIDValue sql.NullString
		if err := rows.Scan(&u.ShortID, &u.OriginalURL, &userIDValue, &u.DeletedFlag); err != nil {
			r.logger.Error("Failed to scan URL row", zap.Error(err))
			return nil, err
		}
		u.UserID = userIDValue.String
		urls = append(urls, u)
	}
	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating URL rows", zap.Error(err))
		return nil, err
	}
	return urls, nil
}

// BatchDelete помечает указанные URL как удалённые
func (r *PostgresRepository) BatchDelete(userID string, ids []string) error {
	query := "UPDATE urls SET is_deleted = TRUE WHERE short_id = ANY($1) AND user_id = $2"
	result, err := r.db.Exec(query, ids, userID)
	if err != nil {
		r.logger.Error("Failed to batch delete URLs",
			zap.String("user_id", userID),
			zap.Strings("ids", ids),
			zap.Error(err))
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", zap.Error(err))
		return err
	}
	r.logger.Info("Batch delete completed",
		zap.String("user_id", userID),
		zap.Int64("rows_affected", rowsAffected))
	return nil
}
