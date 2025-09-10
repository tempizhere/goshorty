package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/app"
	"github.com/tempizhere/goshorty/internal/config"
	"github.com/tempizhere/goshorty/internal/log"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"go.uber.org/zap"
)

// Глобальные переменные для информации о сборке
var buildVersion string
var buildDate string
var buildCommit string

func main() {
	// Выводим информацию о сборке
	printBuildInfo()

	// Получаем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		logger := log.NewLogger()
		logger.Fatal("Failed to initialize configuration", zap.Error(err))
	}

	// Инициализация логгера
	logger := log.NewLogger()

	// Инициализация базы данных
	db, err := app.NewDB(cfg.DatabaseDSN)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer func() {
		if db != nil {
			if closeErr := db.Close(); closeErr != nil {
				logger.Error("Failed to close database", zap.Error(closeErr))
			}
		}
	}()

	// Создаём репозиторий
	var repo repository.Repository
	if cfg.DatabaseDSN != "" && db != nil {
		repo, err = repository.NewPostgresRepository(db, logger)
		if err != nil {
			logger.Fatal("Failed to initialize PostgreSQL repository", zap.Error(err))
		}
		logger.Info("Using PostgreSQL repository")
	} else if cfg.FileStoragePath != "" {
		repo, err = repository.NewFileRepository(cfg.FileStoragePath, logger)
		if err != nil {
			logger.Fatal("Failed to initialize file repository", zap.Error(err))
		}
		logger.Info("Using file repository", zap.String("path", cfg.FileStoragePath))
	} else {
		repo = repository.NewMemoryRepository()
		logger.Info("Using memory repository")
	}

	// Создаём зависимости
	svc := service.NewService(repo, cfg.BaseURL, cfg.JWTSecret)
	appInstance := app.NewApp(svc, db, logger)

	// Создаём маршрутизатор
	r := chi.NewRouter()

	// Применение middleware
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.LoggingMiddleware(logger))
	r.Use(middleware.AuthMiddleware(svc, logger))

	// Регистрируем обработчики
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandlePostURL(w, r)
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleGetURL(w, r)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleJSONShorten(w, r)
	})
	r.Get("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleJSONExpand(w, r)
	})
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandlePing(w, r)
	})
	r.Post("/api/shorten/batch", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleBatchShorten(w, r)
	})
	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleUserURLs(w, r)
	})
	r.Delete("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleBatchDeleteURLs(w, r)
	})

	// Маршрут для статистики с проверкой доверенной подсети
	r.Route("/api/internal", func(r chi.Router) {
		r.Use(middleware.TrustedSubnetMiddleware(cfg.TrustedSubnet, logger))
		r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			appInstance.HandleStats(w, r)
		})
	})

	// Создаём HTTP сервер с настройками для graceful shutdown
	server := &http.Server{
		Addr:         cfg.RunAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	// Создаем контекст, который будет отменен при получении сигнала завершения
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	// Запускаем сервер в горутине
	go func() {
		var err error
		if cfg.EnableHTTPS {
			logger.Info("Starting HTTPS server", zap.String("address", cfg.RunAddr))
			err = server.ListenAndServeTLS("cert.pem", "key.pem")
		} else {
			logger.Info("Starting HTTP server", zap.String("address", cfg.RunAddr))
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Ждем сигнала завершения
	<-ctx.Done()
	logger.Info("Received shutdown signal, starting graceful shutdown...")

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	// Закрываем репозиторий
	if err := repo.Close(); err != nil {
		logger.Error("Failed to close repository", zap.Error(err))
	}

	logger.Info("Graceful shutdown completed")
}

// printBuildInfo выводит информацию о сборке в stdout
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}

	date := buildDate
	if date == "" {
		date = "N/A"
	}

	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}
