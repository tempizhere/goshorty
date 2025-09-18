// Package grpc содержит реализацию gRPC сервера для сервиса сокращения URL
package grpc

import (
	"context"
	"errors"

	"github.com/tempizhere/goshorty/internal/grpc/proto"
	"github.com/tempizhere/goshorty/internal/models"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server реализует gRPC сервер для сервиса сокращения URL
type Server struct {
	proto.UnimplementedShortenerServiceServer
	svc    *service.Service
	db     repository.Database
	logger *zap.Logger
}

// NewServer создаёт новый gRPC сервер
func NewServer(svc *service.Service, db repository.Database, logger *zap.Logger) *Server {
	return &Server{
		svc:    svc,
		db:     db,
		logger: logger,
	}
}

// CreateShortURL обрабатывает создание короткого URL
func (s *Server) CreateShortURL(ctx context.Context, req *proto.CreateShortURLRequest) (*proto.CreateShortURLResponse, error) {
	if req.OriginalURL == "" {
		return nil, status.Error(codes.InvalidArgument, "original URL is required")
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	shortURL, err := s.svc.CreateShortURL(req.OriginalURL, userID)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			return &proto.CreateShortURLResponse{
				ShortURL:  shortURL,
				URLExists: true,
			}, nil
		}
		return nil, s.mapError(err)
	}

	return &proto.CreateShortURLResponse{
		ShortURL:  shortURL,
		URLExists: false,
	}, nil
}

// GetOriginalURL обрабатывает получение оригинального URL
func (s *Server) GetOriginalURL(ctx context.Context, req *proto.GetOriginalURLRequest) (*proto.GetOriginalURLResponse, error) {
	if req.ShortID == "" {
		return nil, status.Error(codes.InvalidArgument, "short ID is required")
	}

	originalURL, exists := s.svc.GetOriginalURL(req.ShortID)
	if !exists {
		u, found := s.svc.Get(req.ShortID)
		if found && u.DeletedFlag {
			return &proto.GetOriginalURLResponse{
				Found:     false,
				IsDeleted: true,
			}, nil
		}
		return &proto.GetOriginalURLResponse{
			Found: false,
		}, nil
	}

	return &proto.GetOriginalURLResponse{
		OriginalURL: originalURL,
		Found:       true,
		IsDeleted:   false,
	}, nil
}

// ShortenURL обрабатывает JSON API для сокращения URL
func (s *Server) ShortenURL(ctx context.Context, req *proto.ShortenURLRequest) (*proto.ShortenURLResponse, error) {
	if req.URL == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	shortURL, err := s.svc.CreateShortURL(req.URL, userID)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			return &proto.ShortenURLResponse{
				Result:    shortURL,
				URLExists: true,
			}, nil
		}
		return nil, s.mapError(err)
	}

	return &proto.ShortenURLResponse{
		Result:    shortURL,
		URLExists: false,
	}, nil
}

// ExpandURL обрабатывает JSON API для получения оригинального URL
func (s *Server) ExpandURL(ctx context.Context, req *proto.ExpandURLRequest) (*proto.ExpandURLResponse, error) {
	if req.ShortID == "" {
		return nil, status.Error(codes.InvalidArgument, "short ID is required")
	}

	originalURL, exists := s.svc.GetOriginalURL(req.ShortID)
	if !exists {
		return &proto.ExpandURLResponse{
			Found: false,
		}, nil
	}

	return &proto.ExpandURLResponse{
		URL:   originalURL,
		Found: true,
	}, nil
}

// Ping проверяет состояние сервиса
func (s *Server) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	if s.db == nil {
		return &proto.PingResponse{DatabaseAvailable: false}, nil
	}

	err := s.db.Ping()
	return &proto.PingResponse{
		DatabaseAvailable: err == nil,
	}, nil
}

// BatchShorten обрабатывает пакетное сокращение URL
func (s *Server) BatchShorten(ctx context.Context, req *proto.BatchShortenRequest) (*proto.BatchShortenResponse, error) {
	if len(req.BatchRequests) == 0 {
		return nil, status.Error(codes.InvalidArgument, "batch requests cannot be empty")
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	requests := make([]models.BatchRequest, len(req.BatchRequests))
	for i, r := range req.BatchRequests {
		requests[i] = models.BatchRequest{
			CorrelationID: r.CorrelationID,
			OriginalURL:   r.OriginalURL,
		}
	}

	responses, err := s.svc.BatchShorten(requests, userID)
	if err != nil {
		if errors.Is(err, repository.ErrURLExists) {
			protoResponses := make([]*proto.BatchResponse, len(responses))
			for i, r := range responses {
				protoResponses[i] = &proto.BatchResponse{
					CorrelationID: r.CorrelationID,
					ShortURL:      r.ShortURL,
				}
			}
			return &proto.BatchShortenResponse{
				BatchResponses: protoResponses,
				HasConflicts:   true,
			}, nil
		}
		return nil, s.mapError(err)
	}

	protoResponses := make([]*proto.BatchResponse, len(responses))
	for i, r := range responses {
		protoResponses[i] = &proto.BatchResponse{
			CorrelationID: r.CorrelationID,
			ShortURL:      r.ShortURL,
		}
	}

	return &proto.BatchShortenResponse{
		BatchResponses: protoResponses,
		HasConflicts:   false,
	}, nil
}

// GetUserURLs возвращает все URL пользователя
func (s *Server) GetUserURLs(ctx context.Context, req *proto.GetUserURLsRequest) (*proto.GetUserURLsResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	urls, err := s.svc.GetURLsByUserID(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user URLs")
	}

	if len(urls) == 0 {
		return &proto.GetUserURLsResponse{UserUrls: []*proto.ShortURLResponse{}}, nil
	}

	protoURLs := make([]*proto.ShortURLResponse, len(urls))
	for i, u := range urls {
		protoURLs[i] = &proto.ShortURLResponse{
			ShortURL:    u.ShortURL,
			OriginalURL: u.OriginalURL,
		}
	}

	return &proto.GetUserURLsResponse{UserUrls: protoURLs}, nil
}

// BatchDeleteURLs удаляет URL пакетно
func (s *Server) BatchDeleteURLs(ctx context.Context, req *proto.BatchDeleteURLsRequest) (*proto.BatchDeleteURLsResponse, error) {
	if len(req.ShortIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "short IDs cannot be empty")
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.svc.BatchDeleteAsync(userID, req.ShortIds)

	return &proto.BatchDeleteURLsResponse{Success: true}, nil
}

// GetStats возвращает статистику сервиса
func (s *Server) GetStats(ctx context.Context, req *proto.GetStatsRequest) (*proto.GetStatsResponse, error) {
	urls, users, err := s.svc.GetStats()
	if err != nil {
		s.logger.Error("Failed to get stats", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get statistics")
	}

	return &proto.GetStatsResponse{
		UrlsCount:  int32(urls),
		UsersCount: int32(users),
	}, nil
}

// getUserIDFromContext извлекает UserID из контекста
func getUserIDFromContext(ctx context.Context) (string, error) {
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		return userID, nil
	}
	return "", status.Error(codes.Unauthenticated, "user not authenticated")
}

// mapError преобразует ошибки бизнес-логики в gRPC статусы
func (s *Server) mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, repository.ErrURLExists):
		return status.Error(codes.AlreadyExists, "URL already exists")
	case errors.Is(err, service.ErrEmptyURL):
		return status.Error(codes.InvalidArgument, "empty URL provided")
	case errors.Is(err, service.ErrEmptyID):
		return status.Error(codes.InvalidArgument, "empty ID provided")
	case err.Error() == "invalid URL":
		return status.Error(codes.InvalidArgument, "invalid URL format")
	default:
		s.logger.Error("Unexpected error", zap.Error(err))
		return status.Error(codes.Internal, "internal server error")
	}
}
