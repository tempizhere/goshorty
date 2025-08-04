package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
)

// contextKey определяет тип для ключей контекста
type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware создаёт middleware для аутентификации пользователей
// Автоматически генерирует JWT токен для новых пользователей и проверяет существующие токены
func AuthMiddleware(svc *service.Service, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID string
			cookie, err := r.Cookie("jwt")
			if err == nil {
				userID, err = svc.ParseJWT(cookie.Value)
				if err != nil {
					logger.Warn("Invalid JWT", zap.Error(err))
				}
			}

			if userID == "" {
				userID, err = svc.GenerateUserID()
				if err != nil {
					logger.Error("Failed to generate user ID", zap.Error(err))
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				token, err := svc.GenerateJWT(userID)
				if err != nil {
					logger.Error("Failed to generate JWT", zap.Error(err))
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     "jwt",
					Value:    token,
					Expires:  time.Now().Add(24 * time.Hour),
					HttpOnly: true,
					Path:     "/",
				})
				logger.Info("Generated new JWT", zap.String("user_id", userID))
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID извлекает UserID из контекста HTTP запроса
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(userIDKey).(string)
	return userID, ok
}
