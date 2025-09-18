// Package grpc содержит интерцепторы для gRPC сервера
package grpc

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// contextKey определяет тип для ключей контекста
type contextKey string

const userIDKey contextKey = "userID"

// AuthInterceptor создаёт интерцептор для аутентификации пользователей
func AuthInterceptor(svc *service.Service, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		publicMethods := map[string]bool{
			"/shortener.v1.ShortenerService/GetOriginalURL": true,
			"/shortener.v1.ShortenerService/ExpandURL":      true,
			"/shortener.v1.ShortenerService/Ping":           true,
		}

		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		var userID string
		var err error

		if authHeaders := md.Get("authorization"); len(authHeaders) > 0 {
			authHeader := authHeaders[0]
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				userID, err = svc.ParseJWT(token)
				if err != nil {
					logger.Warn("Invalid JWT token", zap.Error(err))
				}
			}
		}

		if userID == "" {
			userID, err = svc.GenerateUserID()
			if err != nil {
				logger.Error("Failed to generate user ID", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to generate user ID")
			}

			token, err := svc.GenerateJWT(userID)
			if err != nil {
				logger.Error("Failed to generate JWT", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to generate JWT")
			}

			outgoingMD := metadata.New(map[string]string{
				"authorization": "Bearer " + token,
			})
			if err := grpc.SetHeader(ctx, outgoingMD); err != nil {
				logger.Error("Failed to set response header", zap.Error(err))
			}

			logger.Info("Generated new JWT for gRPC", zap.String("user_id", userID))
		}

		ctx = context.WithValue(ctx, userIDKey, userID)
		return handler(ctx, req)
	}
}

// TrustedSubnetInterceptor создаёт интерцептор для проверки доверенной подсети
func TrustedSubnetInterceptor(trustedSubnet string, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod != "/shortener.v1.ShortenerService/GetStats" {
			return handler(ctx, req)
		}

		if trustedSubnet == "" {
			return nil, status.Error(codes.PermissionDenied, "trusted subnet not configured")
		}

		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "failed to get peer info")
		}

		clientIP := p.Addr.String()
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			clientIP = tcpAddr.IP.String()
		}

		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			logger.Error("Invalid trusted subnet", zap.String("subnet", trustedSubnet), zap.Error(err))
			return nil, status.Error(codes.Internal, "invalid trusted subnet configuration")
		}

		clientIPParsed := net.ParseIP(clientIP)
		if clientIPParsed == nil || !subnet.Contains(clientIPParsed) {
			logger.Warn("Access denied from untrusted IP", zap.String("ip", clientIP))
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}

		return handler(ctx, req)
	}
}

// LoggingInterceptor создаёт интерцептор для логирования gRPC запросов
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		logger.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientIP),
			zap.String("status_code", code.String()),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		return resp, err
	}
}
