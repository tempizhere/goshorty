package main

import (
	"context"
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

func main() {
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
			if err := db.Close(); err != nil {
				logger.Error("Failed to close database", zap.Error(err))
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
	appInstance := app.NewApp(svc, db, cfg)

	// Создаём маршрутизатор
	r := chi.NewRouter()

	// Применение middleware
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.LoggingMiddleware(logger))
	r.Use(middleware.AuthMiddleware(svc, cfg, logger))

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

	// Настраиваем сервер
	srv := &http.Server{
		Addr:    cfg.RunAddr,
		Handler: r,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	logger.Info("Server started", zap.String("addr", cfg.RunAddr))

	<-ctx.Done()
	logger.Info("Shutting down server...")

	// Закрываем базу данных перед shutdown
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database", zap.Error(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}

	logger.Info("Server stopped")
}
