package repository

import (
	sql "database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPostgresRepository(t *testing.T) {
	// Создаём тестовый логгер
	logger := zap.NewNop()

	// Создаём SQL mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Создаём PostgresRepository с реальной *sql.DB
	repo := &PostgresRepository{
		db:     db,
		logger: logger,
	}

	tests := []struct {
		name            string
		setup           func()
		id              string
		url             string
		expectedShortID string
		expectedErr     error
	}{
		{
			name: "Save success",
			setup: func() {
				mock.ExpectQuery("SELECT short_id FROM urls WHERE original_url = \\$1").
					WithArgs("https://example.com").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery("INSERT INTO urls \\(short_id, original_url\\) VALUES \\(\\$1, \\$2\\) ON CONFLICT \\(original_url\\) DO UPDATE SET short_id = urls.short_id RETURNING short_id").
					WithArgs("testID", "https://example.com").
					WillReturnRows(sqlmock.NewRows([]string{"short_id"}).AddRow("testID"))
			},
			id:              "testID",
			url:             "https://example.com",
			expectedShortID: "testID",
			expectedErr:     nil,
		},
		{
			name: "Save error",
			setup: func() {
				mock.ExpectQuery("SELECT short_id FROM urls WHERE original_url = \\$1").
					WithArgs("https://example.com").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery("INSERT INTO urls \\(short_id, original_url\\) VALUES \\(\\$1, \\$2\\) ON CONFLICT \\(original_url\\) DO UPDATE SET short_id = urls.short_id RETURNING short_id").
					WithArgs("testID", "https://example.com").
					WillReturnError(errors.New("db error"))
			},
			id:              "testID",
			url:             "https://example.com",
			expectedShortID: "",
			expectedErr:     errors.New("db error"),
		},
		{
			name: "Save duplicate URL",
			setup: func() {
				mock.ExpectQuery("SELECT short_id FROM urls WHERE original_url = \\$1").
					WithArgs("https://example.com").
					WillReturnRows(sqlmock.NewRows([]string{"short_id"}).AddRow("existingID"))
			},
			id:              "newID",
			url:             "https://example.com",
			expectedShortID: "existingID",
			expectedErr:     ErrURLExists,
		},
		{
			name: "Get not found",
			setup: func() {
				mock.ExpectQuery("SELECT original_url FROM urls WHERE short_id = \\$1").
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			id:              "nonexistent",
			url:             "",
			expectedShortID: "",
			expectedErr:     nil,
		},
		{
			name: "Clear success",
			setup: func() {
				mock.ExpectExec("TRUNCATE TABLE urls RESTART IDENTITY").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			if tt.id != "" {
				if tt.url != "" {
					// Тестируем Save
					shortID, err := repo.Save(tt.id, tt.url)
					assert.Equal(t, tt.expectedErr, err)
					assert.Equal(t, tt.expectedShortID, shortID)
				} else {
					// Тестируем Get
					url, exists := repo.Get(tt.id)
					assert.False(t, exists)
					assert.Equal(t, "", url)
				}
			}

			if tt.name == "Clear success" {
				repo.Clear()
			}

			// Проверяем, что все ожидаемые вызовы мока выполнены
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
