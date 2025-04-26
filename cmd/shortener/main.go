package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/tempizhere/goshorty/internal/app"
	"github.com/tempizhere/goshorty/internal/config"
	"net/http"
)

func main() {
	// Получаем конфигурацию
	cfg := config.NewConfig()

	// Создаём маршрутизатор
	r := chi.NewRouter()

	// Регистрируем обработчики
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		app.HandlePostURL(w, r, cfg)
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		app.HandleGetURL(w, r)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		app.HandleJSONShorten(w, r, cfg)
	})
	r.Get("/api/expand/{id}", func(w http.ResponseWriter, r *http.Request) {
		app.HandleJSONExpand(w, r, cfg)
	})
	err := http.ListenAndServe(cfg.RunAddr, r)
	if err != nil {
		panic(err)
	}
}
