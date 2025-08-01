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

var (
	ErrEmptyURL        = errors.New("empty URL")
	ErrEmptyID         = errors.New("empty ID")
	ErrIDAlreadyExists = errors.New("ID already exists")
	ErrEmptyBatch      = errors.New("empty batch")
	ErrDuplicateCorrID = errors.New("duplicate correlation_id")
	ErrUniqueIDFailed  = errors.New("failed to generate unique ID")
	ErrInvalidToken    = errors.New("invalid token")
)

// Service реализует логику работы с короткими URL
type Service struct {
	repo      repository.Repository
	baseURL   string
	jwtSecret string
}

// NewService создаёт новый экземпляр сервиса
func NewService(repo repository.Repository, baseURL, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		baseURL:   baseURL,
		jwtSecret: jwtSecret,
	}
}

// GenerateShortID генерирует короткий ID
func (s *Service) GenerateShortID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(bytes)
	return encoded[:8], nil
}

// GenerateUserID генерирует уникальный идентификатор пользователя
func (s *Service) GenerateUserID() (string, error) {
	return s.GenerateShortID()
}

// GenerateJWT генерирует JWT с UserID
func (s *Service) GenerateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

// ParseJWT проверяет и извлекает UserID из JWT
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

// CreateShortURLWithID создаёт короткий URL с заданным ID
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
	return strings.TrimRight(s.baseURL, "/") + "/" + shortID, nil
}

// CreateShortURL создаёт короткий URL
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

// BatchShorten создаёт короткие URL для списка запросов
func (s *Service) BatchShorten(reqs []models.BatchRequest, userID string) ([]models.BatchResponse, error) {
	if len(reqs) == 0 {
		return nil, ErrEmptyBatch
	}
	urls := make(map[string]string)
	resp := make([]models.BatchResponse, len(reqs))
	corrIDs := make(map[string]struct{})
	for i, req := range reqs {
		if _, exists := corrIDs[req.CorrelationID]; exists {
			return nil, ErrDuplicateCorrID
		}
		corrIDs[req.CorrelationID] = struct{}{}
		if req.OriginalURL == "" {
			return nil, ErrEmptyURL
		}
		var id string
		var err error
		var shortURL string
		for j := 0; j < 5; j++ {
			id, err = s.GenerateShortID()
			if err != nil {
				return nil, err
			}
			if _, exists := s.repo.Get(id); !exists {
				urls[id] = req.OriginalURL
				shortURL = strings.Join([]string{strings.TrimRight(s.baseURL, "/"), id}, "/")
				break
			}
			if j == 4 {
				return nil, ErrUniqueIDFailed
			}
		}
		resp[i] = models.BatchResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      shortURL,
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

// GetOriginalURL возвращает оригинальный URL по ID
func (s *Service) GetOriginalURL(id string) (string, bool) {
	u, exists := s.repo.Get(id)
	if !exists || u.DeletedFlag {
		return "", false
	}
	return u.OriginalURL, true
}

// Get возвращает полную информацию об URL по ID
func (s *Service) Get(id string) (models.URL, bool) {
	return s.repo.Get(id)
}

// GetURLsByUserID возвращает все URL, связанные с пользователем
func (s *Service) GetURLsByUserID(userID string) ([]models.ShortURLResponse, error) {
	urls, err := s.repo.GetURLsByUserID(userID)
	if err != nil {
		return nil, err
	}
	resp := make([]models.ShortURLResponse, len(urls))
	for i, u := range urls {
		resp[i] = models.ShortURLResponse{
			ShortURL:    strings.TrimRight(s.baseURL, "/") + "/" + u.ShortID,
			OriginalURL: u.OriginalURL,
		}
	}
	return resp, nil
}

// BatchDelete помечает указанные URL как удалённые
func (s *Service) BatchDelete(userID string, ids []string) error {
	return s.repo.BatchDelete(userID, ids)
}

// BatchDeleteAsync асинхронно помечает указанные URL как удалённые
func (s *Service) BatchDeleteAsync(userID string, ids []string) {
	go func() {
		s.BatchDelete(userID, ids)
	}()
}
