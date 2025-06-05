package models

type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// Структура для хранения URL с UserID
type URL struct {
	ShortID     string `json:"short_id"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

// Структура для ответа хендлера /api/user/urls
type ShortURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
