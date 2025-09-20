// Package proto содержит определения типов для gRPC сервиса сокращения URL
package proto

// CreateShortURLRequest представляет запрос на создание короткого URL
type CreateShortURLRequest struct {
	OriginalURL string `json:"original_url"`
}

// CreateShortURLResponse представляет ответ с созданным коротким URL
type CreateShortURLResponse struct {
	ShortURL  string `json:"short_url"`
	URLExists bool   `json:"url_exists"`
}

// GetOriginalURLRequest представляет запрос на получение оригинального URL
type GetOriginalURLRequest struct {
	ShortID string `json:"short_id"`
}

// GetOriginalURLResponse представляет ответ с оригинальным URL
type GetOriginalURLResponse struct {
	OriginalURL string `json:"original_url"`
	Found       bool   `json:"found"`
	IsDeleted   bool   `json:"is_deleted"`
}

// ShortenURLRequest представляет JSON запрос на сокращение URL
type ShortenURLRequest struct {
	URL string `json:"url"`
}

// ShortenURLResponse представляет JSON ответ с коротким URL
type ShortenURLResponse struct {
	Result    string `json:"result"`
	URLExists bool   `json:"url_exists"`
}

// ExpandURLRequest представляет запрос на получение оригинального URL через API
type ExpandURLRequest struct {
	ShortID string `json:"short_id"`
}

// ExpandURLResponse представляет ответ с оригинальным URL через API
type ExpandURLResponse struct {
	URL   string `json:"url"`
	Found bool   `json:"found"`
}

// PingRequest представляет запрос проверки состояния
type PingRequest struct{}

// PingResponse представляет ответ проверки состояния
type PingResponse struct {
	DatabaseAvailable bool `json:"database_available"`
}

// BatchRequest представляет элемент пакетного запроса
type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchResponse представляет элемент пакетного ответа
type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// BatchShortenRequest представляет запрос пакетного сокращения
type BatchShortenRequest struct {
	BatchRequests []*BatchRequest `json:"batch_requests"`
}

// BatchShortenResponse представляет ответ пакетного сокращения
type BatchShortenResponse struct {
	BatchResponses []*BatchResponse `json:"batch_responses"`
	HasConflicts   bool             `json:"has_conflicts"`
}

// GetUserURLsRequest представляет запрос на получение URL пользователя
type GetUserURLsRequest struct{}

// ShortURLResponse представляет информацию о коротком URL
type ShortURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// GetUserURLsResponse представляет ответ со списком URL пользователя
type GetUserURLsResponse struct {
	UserUrls []*ShortURLResponse `json:"user_urls"`
}

// BatchDeleteURLsRequest представляет запрос пакетного удаления URL
type BatchDeleteURLsRequest struct {
	ShortIds []string `json:"short_ids"`
}

// BatchDeleteURLsResponse представляет ответ пакетного удаления URL
type BatchDeleteURLsResponse struct {
	Success bool `json:"success"`
}

// GetStatsRequest представляет запрос статистики
type GetStatsRequest struct{}

// GetStatsResponse представляет ответ со статистикой
type GetStatsResponse struct {
	UrlsCount  int32 `json:"urls_count"`
	UsersCount int32 `json:"users_count"`
}
