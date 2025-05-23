package repository

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPostgresRepository(t *testing.T) {
	// Создаём контроллер gomock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Создаём мок для Database
	mockDB := NewMockDatabase(ctrl)

	// Создаём тестовый логгер
	logger := zap.NewNop()

	// Создаём PostgresRepository
	repo := &PostgresRepository{
		db:     mockDB,
		logger: logger,
	}

	tests := []struct {
		name        string
		setup       func()
		id          string
		url         string
		expectedErr error
	}{
		{
			name: "Save success",
			setup: func() {
				mockDB.EXPECT().Exec("INSERT INTO urls (short_id, original_url) VALUES ($1, $2)", "testID", "https://example.com").Return(nil, nil)
				mockDB.EXPECT().Begin().Times(0)
			},
			id:          "testID",
			url:         "https://example.com",
			expectedErr: nil,
		},
		{
			name: "Save error",
			setup: func() {
				mockDB.EXPECT().Exec("INSERT INTO urls (short_id, original_url) VALUES ($1, $2)", "testID", "https://example.com").Return(nil, errors.New("db error"))
				mockDB.EXPECT().Begin().Times(0)
			},
			id:          "testID",
			url:         "https://example.com",
			expectedErr: errors.New("db error"),
		},
		{
			name: "Get expectation",
			setup: func() {
				mockDB.EXPECT().QueryRow("SELECT original_url FROM urls WHERE short_id = $1", gomock.Any()).Times(0)
				mockDB.EXPECT().Begin().Times(0)
			},
		},
		{
			name: "Clear success",
			setup: func() {
				mockDB.EXPECT().Exec("TRUNCATE TABLE urls RESTART IDENTITY").Return(nil, nil)
				mockDB.EXPECT().Begin().Times(0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			if tt.id != "" {
				err := repo.Save(tt.id, tt.url)
				assert.Equal(t, tt.expectedErr, err)
			}

			if tt.name == "Clear success" {
				repo.Clear()
			}
		})
	}
}
