// Package service реализует бизнес-логику сервиса сокращения URL.
// Содержит основные операции по созданию, получению и управлению короткими URL,
// а также функции аутентификации и работы с JWT токенами.
package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
)

// ErrEmptyURL возвращается при попытке создать короткий URL из пустой строки
var ErrEmptyURL = errors.New("empty URL")

// ErrEmptyID возвращается при попытке создать URL с пустым ID
var ErrEmptyID = errors.New("empty ID")

// ErrIDAlreadyExists возвращается при попытке создать URL с уже существующим ID
var ErrIDAlreadyExists = errors.New("ID already exists")

// ErrEmptyBatch возвращается при попытке обработать пустой пакет запросов
var ErrEmptyBatch = errors.New("empty batch")

// ErrDuplicateCorrID возвращается при обнаружении дублирующихся correlation_id в пакете
var ErrDuplicateCorrID = errors.New("duplicate correlation_id")

// ErrUniqueIDFailed возвращается при неудачной попытке генерации уникального ID
var ErrUniqueIDFailed = errors.New("failed to generate unique ID")

// ErrInvalidToken возвращается при неверном или истёкшем JWT токене
var ErrInvalidToken = errors.New("invalid token")

// Service реализует бизнес-логику работы с короткими URL
type Service struct {
	repo      repository.Repository // Репозиторий для работы с данными
	baseURL   string                // Базовый URL для генерации коротких ссылок
	jwtSecret string                // Секретный ключ для подписи JWT токенов
}

// NewService создаёт новый экземпляр сервиса с указанным репозиторием, базовым URL и секретным ключом JWT
func NewService(repo repository.Repository, baseURL, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		baseURL:   baseURL,
		jwtSecret: jwtSecret,
	}
}

// GenerateShortID генерирует случайный короткий ID длиной 8 символов в base64url кодировке
func (s *Service) GenerateShortID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(bytes)
	return encoded[:8], nil
}

// GenerateUserID генерирует уникальный идентификатор пользователя, используя тот же алгоритм, что и для коротких ID
func (s *Service) GenerateUserID() (string, error) {
	return s.GenerateShortID()
}

// GenerateJWT генерирует JWT токен с указанным UserID и сроком действия 24 часа
func (s *Service) GenerateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

// ParseJWT проверяет подпись JWT токена и извлекает UserID из payload
func (s *Service) ParseJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidToken
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", ErrInvalidToken
	}
	return userID, nil
}

// CreateShortURLWithID создаёт короткий URL с заданным ID для указанного пользователя
func (s *Service) CreateShortURLWithID(originalURL, id, userID string) (string, error) {
	if originalURL == "" {
		return "", ErrEmptyURL
	}
	if id == "" {
		return "", ErrEmptyID
	}
	if _, exists := s.repo.Get(id); exists {
		return "", ErrIDAlreadyExists
	}
	shortID, err := s.repo.Save(id, originalURL, userID)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			return strings.TrimRight(s.baseURL, "/") + "/" + shortID, repository.ErrURLExists
		}
		return "", err
	}
	// Используем простое конкатенацию вместо strings.Builder для коротких строк
	return strings.TrimRight(s.baseURL, "/") + "/" + shortID, nil
}

// CreateShortURL создаёт короткий URL с автоматически сгенерированным ID для указанного пользователя
func (s *Service) CreateShortURL(originalURL, userID string) (string, error) {
	var id string
	var err error
	for i := 0; i < 5; i++ {
		id, err = s.GenerateShortID()
		if err != nil {
			return "", err
		}
		shortURL, err := s.CreateShortURLWithID(originalURL, id, userID)
		if err == nil {
			return shortURL, nil
		}
		if errors.Is(err, repository.ErrURLExists) {
			return shortURL, repository.ErrURLExists
		}
		if errors.Is(err, ErrIDAlreadyExists) {
			continue
		}
		return "", err
	}
	return "", errors.New("failed to generate unique ID")
}

// BatchShorten создаёт короткие URL для списка запросов в пакетном режиме для указанного пользователя
func (s *Service) BatchShorten(reqs []models.BatchRequest, userID string) ([]models.BatchResponse, error) {
	if len(reqs) == 0 {
		return nil, ErrEmptyBatch
	}
	urls := make(map[string]string, len(reqs))
	resp := make([]models.BatchResponse, 0, len(reqs))
	corrIDs := make(map[string]struct{}, len(reqs))

	// Предварительно вычисляем базовый URL
	baseURL := strings.TrimRight(s.baseURL, "/")
	baseURLLen := len(baseURL)

	for _, req := range reqs {
		if _, exists := corrIDs[req.CorrelationID]; exists {
			return nil, ErrDuplicateCorrID
		}
		corrIDs[req.CorrelationID] = struct{}{}
		if req.OriginalURL == "" {
			return nil, ErrEmptyURL
		}
		var id string
		var err error
		for j := 0; j < 5; j++ {
			id, err = s.GenerateShortID()
			if err != nil {
				return nil, err
			}
			if _, exists := s.repo.Get(id); !exists {
				urls[id] = req.OriginalURL
				// Формирование URL с использованием append для экономии памяти
				shortURL := make([]byte, 0, baseURLLen+9) // baseURL + "/" + 8-char id
				shortURL = append(shortURL, baseURL...)
				shortURL = append(shortURL, '/')
				shortURL = append(shortURL, id...)
				resp = append(resp, models.BatchResponse{
					CorrelationID: req.CorrelationID,
					ShortURL:      string(shortURL),
				})
				break
			}
			if j == 4 {
				return nil, ErrUniqueIDFailed
			}
		}
	}
	if err := s.repo.BatchSave(urls, userID); err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			return resp, repository.ErrURLExists
		}
		return nil, err
	}
	return resp, nil
}

// GetOriginalURL возвращает оригинальный URL по короткому ID, учитывая флаг удаления
func (s *Service) GetOriginalURL(id string) (string, bool) {
	u, exists := s.repo.Get(id)
	if !exists || u.DeletedFlag {
		return "", false
	}
	return u.OriginalURL, true
}

// Get возвращает полную информацию об URL по короткому ID
func (s *Service) Get(id string) (models.URL, bool) {
	return s.repo.Get(id)
}

// GetURLsByUserID возвращает все URL, созданные указанным пользователем, в формате для API ответа
func (s *Service) GetURLsByUserID(userID string) ([]models.ShortURLResponse, error) {
	urls, err := s.repo.GetURLsByUserID(userID)
	if err != nil {
		return nil, err
	}
	resp := make([]models.ShortURLResponse, 0, len(urls))

	// Предварительно вычисляем базовый URL
	baseURL := strings.TrimRight(s.baseURL, "/")
	baseURLLen := len(baseURL)

	for _, u := range urls {
		// Формирование URL с использованием append для экономии памяти
		shortURL := make([]byte, 0, baseURLLen+len(u.ShortID)+1)
		shortURL = append(shortURL, baseURL...)
		shortURL = append(shortURL, '/')
		shortURL = append(shortURL, u.ShortID...)
		resp = append(resp, models.ShortURLResponse{
			ShortURL:    string(shortURL),
			OriginalURL: u.OriginalURL,
		})
	}
	return resp, nil
}

// BatchDelete помечает указанные URL как удалённые для указанного пользователя
func (s *Service) BatchDelete(userID string, ids []string) error {
	return s.repo.BatchDelete(userID, ids)
}

// BatchDeleteAsync асинхронно помечает указанные URL как удалённые для указанного пользователя
func (s *Service) BatchDeleteAsync(userID string, ids []string) {
	go func() {
		if err := s.BatchDelete(userID, ids); err != nil {
			_ = err
		}
	}()
}
