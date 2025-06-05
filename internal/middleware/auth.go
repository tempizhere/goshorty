package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/tempizhere/goshorty/internal/config"
	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
)

// UserIDKey для хранения UserID в контексте
type UserIDKey struct{}

// AuthMiddleware проверяет и устанавливает куку с JWT
func AuthMiddleware(svc *service.Service, cfg *config.Config, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID string
			var err error

			// Проверяем UserID из контекста
			if id, ok := r.Context().Value(UserIDKey{}).(string); ok && id != "" {
				userID = id
			} else {
				// Проверяем куку с JWT
				cookie, err := r.Cookie("jwt_token")
				if err == nil && cookie != nil {
					userID, err = svc.ParseJWT(cookie.Value)
					if err != nil {
						logger.Warn("Invalid JWT token", zap.Error(err))
					}
				}
			}

			// Требуем userID для защищённых маршрутов
			protectedRoutes := map[string]bool{
				"/api/user/urls":     true,
				"/api/shorten/batch": true,
			}
			if userID == "" && protectedRoutes[r.URL.Path] && r.Method == http.MethodGet {
				// Генерируем новый UserID для GET /api/user/urls
				userID, err = svc.GenerateUserID()
				if err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				token, err := svc.GenerateJWT(userID)
				if err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     "jwt_token",
					Value:    token,
					Expires:  time.Now().Add(cfg.CookieTTL),
					Path:     "/",
					HttpOnly: true,
				})
			} else if userID == "" && protectedRoutes[r.URL.Path] {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Если userID не установлен, генерируем новый
			if userID == "" {
				userID, err = svc.GenerateUserID()
				if err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				token, err := svc.GenerateJWT(userID)
				if err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     "jwt_token",
					Value:    token,
					Expires:  time.Now().Add(cfg.CookieTTL),
					Path:     "/",
					HttpOnly: true,
				})
			}

			// Добавляем UserID в контекст
			ctx := context.WithValue(r.Context(), UserIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID извлекает UserID из контекста
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey{}).(string)
	return userID, ok
}
