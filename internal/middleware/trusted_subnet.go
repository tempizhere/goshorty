// Package middleware содержит HTTP middleware для обработки запросов.
// Включает аутентификацию, логирование, сжатие ответов и проверку доверенных подсетей.
package middleware

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

// TrustedSubnetMiddleware создаёт middleware для проверки IP-адреса в доверенной подсети
// Проверяет заголовок X-Real-IP и сравнивает с CIDR-нотацией trusted_subnet
func TrustedSubnetMiddleware(trustedSubnet string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если trusted_subnet пустой, запрещаем доступ
			if trustedSubnet == "" {
				logger.Warn("Access denied: trusted_subnet is empty",
					zap.String("method", r.Method),
					zap.String("uri", r.RequestURI),
					zap.String("remote_addr", r.RemoteAddr))
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Получаем IP-адрес из заголовка X-Real-IP
			clientIP := r.Header.Get("X-Real-IP")
			if clientIP == "" {
				logger.Warn("Access denied: X-Real-IP header is missing",
					zap.String("method", r.Method),
					zap.String("uri", r.RequestURI),
					zap.String("remote_addr", r.RemoteAddr))
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Парсим IP-адрес клиента
			ip := net.ParseIP(clientIP)
			if ip == nil {
				logger.Warn("Access denied: invalid IP address in X-Real-IP header",
					zap.String("method", r.Method),
					zap.String("uri", r.RequestURI),
					zap.String("client_ip", clientIP),
					zap.String("remote_addr", r.RemoteAddr))
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Парсим CIDR-нотацию
			_, network, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				logger.Error("Invalid trusted_subnet CIDR",
					zap.String("trusted_subnet", trustedSubnet),
					zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Проверяем, входит ли IP в доверенную подсеть
			if !network.Contains(ip) {
				logger.Warn("Access denied: IP not in trusted subnet",
					zap.String("method", r.Method),
					zap.String("uri", r.RequestURI),
					zap.String("client_ip", clientIP),
					zap.String("trusted_subnet", trustedSubnet),
					zap.String("remote_addr", r.RemoteAddr))
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// IP входит в доверенную подсеть, разрешаем доступ
			logger.Info("Access granted: IP in trusted subnet",
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.String("client_ip", clientIP),
				zap.String("trusted_subnet", trustedSubnet))

			next.ServeHTTP(w, r)
		})
	}
}
