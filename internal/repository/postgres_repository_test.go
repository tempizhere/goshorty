package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/tempizhere/goshorty/internal/models"
	"go.uber.org/zap"
)

func TestPostgresRepository(t *testing.T) {
	logger := zap.NewNop()
	userID := "test_user"

	// Создаём SQL mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := &PostgresRepository{
		db:     db,
		logger: logger,
	}

	tests := []struct {
		name            string
		setup           func()
		id              string
		url             string
		userID          string
		expectedShortID string
		expectedErr     error
		execute         func(*PostgresRepository) error
	}{
		{
			name: "Save success",
			setup: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT short_id FROM urls WHERE original_url = $1")).
					WithArgs("https://example.com").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO UPDATE SET short_id = urls.short_id RETURNING short_id")).
					WithArgs("testID", "https://example.com", userID).
					WillReturnRows(sqlmock.NewRows([]string{"short_id"}).AddRow("testID"))
			},
			id:              "testID",
			url:             "https://example.com",
			userID:          userID,
			expectedShortID: "testID",
			expectedErr:     nil,
		},
		{
			name: "Save error",
			setup: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT short_id FROM urls WHERE original_url = $1")).
					WithArgs("https://example.com").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO UPDATE SET short_id = urls.short_id RETURNING short_id")).
					WithArgs("testID", "https://example.com", userID).
					WillReturnError(errors.New("db error"))
			},
			id:              "testID",
			url:             "https://example.com",
			userID:          userID,
			expectedShortID: "",
			expectedErr:     errors.New("db error"),
		},
		{
			name: "Save duplicate URL",
			setup: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT short_id FROM urls WHERE original_url = $1")).
					WithArgs("https://example.com").
					WillReturnRows(sqlmock.NewRows([]string{"short_id"}).AddRow("existingID"))
			},
			id:              "newID",
			url:             "https://example.com",
			userID:          userID,
			expectedShortID: "existingID",
			expectedErr:     ErrURLExists,
		},
		{
			name: "Get not found",
			setup: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT original_url FROM urls WHERE short_id = $1")).
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			id:              "nonexistent",
			url:             "",
			userID:          userID,
			expectedShortID: "",
			expectedErr:     nil,
		},
		{
			name: "Clear success",
			setup: func() {
				mock.ExpectExec(regexp.QuoteMeta("TRUNCATE TABLE urls RESTART IDENTITY")).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
		},
		{
			name: "BatchSave success",
			setup: func() {
				mock.ExpectExec(regexp.QuoteMeta("TRUNCATE TABLE urls RESTART IDENTITY")).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectBegin()
				query := regexp.QuoteMeta("INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO UPDATE SET short_id = urls.short_id RETURNING short_id")
				// Ожидаем запросы в любом порядке
				mock.ExpectQuery(query).
					WithArgs(sqlmock.AnyArg(), "https://example.com", userID).
					WillReturnRows(sqlmock.NewRows([]string{"short_id"}).AddRow("id1"))
				mock.ExpectQuery(query).
					WithArgs(sqlmock.AnyArg(), "https://test.com", userID).
					WillReturnRows(sqlmock.NewRows([]string{"short_id"}).AddRow("id2"))
				mock.ExpectCommit()
			},
			id:  "batch",
			url: "https://example.com,https://test.com",
			execute: func(r *PostgresRepository) error {
				r.Clear()
				urls := map[string]string{
					"id1": "https://example.com",
					"id2": "https://test.com",
				}
				return r.BatchSave(urls, userID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			if tt.execute != nil {
				// Выполняем пользовательскую функцию
				err := tt.execute(repo)
				assert.Equal(t, tt.expectedErr, err)
			} else if tt.id != "" && tt.id != "batch" {
				if tt.url != "" {
					// Тестируем Save
					shortID, err := repo.Save(tt.id, tt.url, tt.userID)
					assert.Equal(t, tt.expectedErr, err)
					assert.Equal(t, tt.expectedShortID, shortID)
				} else {
					// Тестируем Get
					url, exists := repo.Get(tt.id)
					assert.False(t, exists)
					assert.Equal(t, "", url)
				}
			} else if tt.name == "Clear success" {
				repo.Clear()
			}

			// Проверяем, что все ожидаемые вызовы мока выполнены
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetURLsByUserID(t *testing.T) {
	logger := zap.NewNop()
	userID := "test_user"

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := &PostgresRepository{db: db, logger: logger}

	tests := []struct {
		name         string
		userID       string
		setup        func()
		expectedURLs []models.URL
		expectedErr  error
	}{
		{
			name:   "Success with URLs",
			userID: userID,
			setup: func() {
				rows := sqlmock.NewRows([]string{"short_id", "original_url", "user_id"}).
					AddRow("id1", "https://example.com", userID).
					AddRow("id2", "https://test.com", userID)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT short_id, original_url, user_id FROM urls WHERE user_id = $1")).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			expectedURLs: []models.URL{
				{ShortID: "id1", OriginalURL: "https://example.com", UserID: userID},
				{ShortID: "id2", OriginalURL: "https://test.com", UserID: userID},
			},
			expectedErr: nil,
		},
		{
			name:   "No URLs",
			userID: "unknown_user",
			setup: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT short_id, original_url, user_id FROM urls WHERE user_id = $1")).
					WithArgs("unknown_user").
					WillReturnRows(sqlmock.NewRows([]string{"short_id", "original_url", "user_id"}))
			},
			expectedURLs: []models.URL{},
			expectedErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			urls, err := repo.GetURLsByUserID(tt.userID)
			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedURLs, urls)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
