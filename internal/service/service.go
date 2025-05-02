package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/tempizhere/goshorty/internal/repository"
	"strings"
)

// Service реализует логику работы с короткими URL
type Service struct {
	repo    repository.Repository
	baseURL string // придуманное имя
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
		return "", errors.New("empty URL")
	}
	if id == "" {
		return "", errors.New("empty ID")
	}
	if _, exists := s.repo.Get(id); exists {
		return "", errors.New("ID already exists")
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
		if !strings.Contains(err.Error(), "ID already exists") {
			return "", err
		}
	}
	return "", errors.New("failed to generate unique ID")
}

// GetOriginalURL возвращает оригинальный URL по ID
func (s *Service) GetOriginalURL(id string) (string, bool) {
	return s.repo.Get(id)
}

// ExtractIDFromShortURL извлекает ID из короткого URL
func (s *Service) ExtractIDFromShortURL(shortURL string) string {
	return shortURL[strings.LastIndex(shortURL, "/")+1:]
}
