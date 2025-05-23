package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
)

var (
	ErrEmptyURL        = errors.New("empty URL")
	ErrEmptyID         = errors.New("empty ID")
	ErrIDAlreadyExists = errors.New("ID already exists")
)

// Service реализует логику работы с короткими URL
type Service struct {
	repo    repository.Repository
	baseURL string
}

// NewService создаёт новый экземпляр Service
func NewService(repo repository.Repository, baseURL string) *Service {
	return &Service{
		repo:    repo,
		baseURL: baseURL,
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

// CreateShortURLWithID создаёт короткий URL с заданным ID
func (s *Service) CreateShortURLWithID(originalURL, id string) (string, error) {
	if originalURL == "" {
		return "", ErrEmptyURL
	}
	if id == "" {
		return "", ErrEmptyID
	}
	if _, exists := s.repo.Get(id); exists {
		return "", ErrIDAlreadyExists
	}
	err := s.repo.Save(id, originalURL)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s.baseURL, "/") + "/" + id, nil
}

// CreateShortURL создаёт короткий URL
func (s *Service) CreateShortURL(originalURL string) (string, error) {
	var id string
	var err error
	for i := 0; i < 5; i++ {
		id, err = s.GenerateShortID()
		if err != nil {
			return "", err
		}
		shortURL, err := s.CreateShortURLWithID(originalURL, id)
		if err == nil {
			return shortURL, nil
		}
		if errors.Is(err, ErrIDAlreadyExists) {
			continue
		}
		return "", err
	}
	return "", errors.New("failed to generate unique ID")
}

// BatchShorten создаёт короткие URL для списка запросов
func (s *Service) BatchShorten(reqs []models.BatchRequest) ([]models.BatchResponse, error) {
	if len(reqs) == 0 {
		return nil, errors.New("empty batch")
	}
	urls := make(map[string]string)
	resp := make([]models.BatchResponse, len(reqs))
	corrIDs := make(map[string]struct{})
	for i, req := range reqs {
		if _, exists := corrIDs[req.CorrelationID]; exists {
			return nil, errors.New("duplicate correlation_id")
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
				break
			}
			if j == 4 {
				return nil, errors.New("failed to generate unique ID")
			}
		}
		urls[id] = req.OriginalURL
		resp[i] = models.BatchResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      strings.TrimRight(s.baseURL, "/") + "/" + id,
		}
	}
	if err := s.repo.BatchSave(urls); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOriginalURL возвращает оригинальный URL по ID
func (s *Service) GetOriginalURL(id string) (string, bool) {
	return s.repo.Get(id)
}

// ExtractIDFromShortURL извлекает ID из короткого URL
func (s *Service) ExtractIDFromShortURL(shortURL string) string {
	return shortURL[strings.LastIndex(shortURL, "/")+1:]
}
