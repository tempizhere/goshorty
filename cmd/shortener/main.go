package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

	// Создаём HTTP сервер с настройками для graceful shutdown
	server := &http.Server{
		Addr:         cfg.RunAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Канал для получения сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Запускаем сервер в горутине
	go func() {
		if cfg.EnableHTTPS {
			logger.Info("Starting HTTPS server", zap.String("address", cfg.RunAddr))
			if err := server.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
				logger.Fatal("Failed to start HTTPS server", zap.Error(err))
			}
		} else {
			logger.Info("Starting HTTP server", zap.String("address", cfg.RunAddr))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal("Failed to start HTTP server", zap.Error(err))
			}
		}
	}()

	// Ожидаем сигнал завершения
	sig := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	logger.Info("Starting graceful shutdown...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем сервер
	if err := server.Shutdown(ctx); err != nil {
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
