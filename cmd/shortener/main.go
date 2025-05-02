package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/app"
	"github.com/tempizhere/goshorty/internal/config"
	"github.com/tempizhere/goshorty/internal/log"
	"github.com/tempizhere/goshorty/internal/middleware"
	"github.com/tempizhere/goshorty/internal/repository"
	"github.com/tempizhere/goshorty/internal/service"
	"net/http"
)

func main() {
	// Получаем конфигурацию
	cfg := config.NewConfig()
	repo := repository.NewMemoryRepository()
	svc := service.NewService(repo, cfg.BaseURL)
	appInstance := app.NewApp(svc)

	// Создаём маршрутизатор
	r := chi.NewRouter()

	// Инициализация логгера
	logger := log.NewLogger()

	// Применение middleware для логирования
	r.Use(middleware.LoggingMiddleware(logger))

	// Регистрируем обработчики
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandlePostURL(w, r)
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleGetURL(w, r)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleJSONShorten(w, r)
	})
	r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
		appInstance.HandleJSONExpand(w, r)
	})
	err := http.ListenAndServe(cfg.RunAddr, r)
	if err != nil {
		panic(err)
	}
}
