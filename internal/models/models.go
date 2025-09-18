// Package models содержит структуры данных для сервиса сокращения URL.
// Определяет модели для запросов и ответов API, включая пакетные операции и пользовательские URL.
package models

// BatchRequest представляет запрос на пакетное сокращение URL
type BatchRequest struct {
	CorrelationID string `json:"correlation_id"` // Уникальный идентификатор для связи запроса и ответа
	OriginalURL   string `json:"original_url"`   // Оригинальный URL для сокращения
}

// BatchResponse представляет ответ на пакетное сокращение URL
type BatchResponse struct {
	CorrelationID string `json:"correlation_id"` // Уникальный идентификатор для связи запроса и ответа
	ShortURL      string `json:"short_url"`      // Сокращённый URL
}

// URL представляет структуру URL в системе
type URL struct {
	ShortID     string `json:"short_id"`                   // Короткий идентификатор URL
	OriginalURL string `json:"original_url"`               // Оригинальный URL
	UserID      string `json:"user_id"`                    // Идентификатор пользователя, создавшего URL
	DeletedFlag bool   `json:"is_deleted" db:"is_deleted"` // Флаг удаления URL
}

// ShortURLResponse представляет ответ с информацией о сокращённом URL
type ShortURLResponse struct {
	ShortURL    string `json:"short_url"`    // Сокращённый URL
	OriginalURL string `json:"original_url"` // Оригинальный URL
}

// StatsResponse представляет ответ с статистикой сервиса
type StatsResponse struct {
	URLs  int `json:"urls"`  // количество сокращённых URL в сервисе
	Users int `json:"users"` // количество пользователей в сервисе
}
