// Package proto содержит интерфейс gRPC сервиса сокращения URL
package proto

import (
	"context"

	"google.golang.org/grpc"
)

// ShortenerServiceServer представляет интерфейс gRPC сервиса
type ShortenerServiceServer interface {
	CreateShortURL(ctx context.Context, req *CreateShortURLRequest) (*CreateShortURLResponse, error)
	GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error)
	ShortenURL(ctx context.Context, req *ShortenURLRequest) (*ShortenURLResponse, error)
	ExpandURL(ctx context.Context, req *ExpandURLRequest) (*ExpandURLResponse, error)
	Ping(ctx context.Context, req *PingRequest) (*PingResponse, error)
	BatchShorten(ctx context.Context, req *BatchShortenRequest) (*BatchShortenResponse, error)
	GetUserURLs(ctx context.Context, req *GetUserURLsRequest) (*GetUserURLsResponse, error)
	BatchDeleteURLs(ctx context.Context, req *BatchDeleteURLsRequest) (*BatchDeleteURLsResponse, error)
	GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsResponse, error)
}

// UnimplementedShortenerServiceServer предоставляет базовую реализацию интерфейса
type UnimplementedShortenerServiceServer struct{}

// CreateShortURL предоставляет базовую реализацию метода создания короткого URL
func (UnimplementedShortenerServiceServer) CreateShortURL(ctx context.Context, req *CreateShortURLRequest) (*CreateShortURLResponse, error) {
	return nil, nil
}

// GetOriginalURL предоставляет базовую реализацию метода получения оригинального URL
func (UnimplementedShortenerServiceServer) GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error) {
	return nil, nil
}

// ShortenURL предоставляет базовую реализацию JSON API для сокращения URL
func (UnimplementedShortenerServiceServer) ShortenURL(ctx context.Context, req *ShortenURLRequest) (*ShortenURLResponse, error) {
	return nil, nil
}

// ExpandURL предоставляет базовую реализацию JSON API для получения оригинального URL
func (UnimplementedShortenerServiceServer) ExpandURL(ctx context.Context, req *ExpandURLRequest) (*ExpandURLResponse, error) {
	return nil, nil
}

// Ping предоставляет базовую реализацию проверки состояния сервиса
func (UnimplementedShortenerServiceServer) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	return nil, nil
}

// BatchShorten предоставляет базовую реализацию пакетного сокращения URL
func (UnimplementedShortenerServiceServer) BatchShorten(ctx context.Context, req *BatchShortenRequest) (*BatchShortenResponse, error) {
	return nil, nil
}

// GetUserURLs предоставляет базовую реализацию получения URL пользователя
func (UnimplementedShortenerServiceServer) GetUserURLs(ctx context.Context, req *GetUserURLsRequest) (*GetUserURLsResponse, error) {
	return nil, nil
}

// BatchDeleteURLs предоставляет базовую реализацию пакетного удаления URL
func (UnimplementedShortenerServiceServer) BatchDeleteURLs(ctx context.Context, req *BatchDeleteURLsRequest) (*BatchDeleteURLsResponse, error) {
	return nil, nil
}

// GetStats предоставляет базовую реализацию получения статистики сервиса
func (UnimplementedShortenerServiceServer) GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsResponse, error) {
	return nil, nil
}

// RegisterShortenerServiceServer регистрирует реализацию сервиса в gRPC сервере
func RegisterShortenerServiceServer(s *grpc.Server, srv ShortenerServiceServer) {
	// В реальном проекте это было бы автоматически сгенерировано protoc
	// Здесь заглушка для демонстрации
}
