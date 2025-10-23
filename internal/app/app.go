// Package app содержит HTTP-обработчики и вспомогательные элементы веб-приложения.
package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/auth"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/services"
)

// Handler - структура сервиса
type Handler struct {
	urlService services.URLService
}

// InitHandlers Инициализация хендлеров
func InitHandlers(urlService services.URLService) *chi.Mux {
	h := &Handler{urlService: urlService}
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	r.Use(h.GzipMiddleware)
	r.Use(logger.RequestLogger)

	r.Post("/", h.PostHandler)
	r.Post("/api/shorten", h.PostShortenHandler)
	r.Post("/api/shorten/batch", h.PostShortenHandlerBatch)
	r.Get("/{id}", h.GetHandler)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
	r.Get("/ping", h.GetDBPing)

	return r
}
