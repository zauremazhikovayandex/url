package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/services"
)

type Handler struct {
	urlService services.URLService
}

func InitHandlers(urlService services.URLService) *chi.Mux {
	h := &Handler{urlService: urlService}
	r := chi.NewRouter()
	r.Use(h.GzipMiddleware)
	r.Use(logger.RequestLogger)

	r.Post("/", h.PostHandler)
	r.Post("/api/shorten", h.PostShortenHandler)
	r.Post("/api/shorten/batch", h.PostShortenHandlerBatch)
	r.Get("/{id}", h.GetHandler)
	r.Get("/ping", h.GetDBPing)

	return r
}
