package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/logger"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(GzipMiddleware)
	r.Use(logger.RequestLogger)

	r.Post("/", PostHandler)
	r.Post("/api/shorten", PostShortenHandler)
	r.Get("/{id}", GetHandler)
	r.Get("/ping", GetDBPing)
	return r
}
