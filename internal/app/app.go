package app

import (
	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()

	r.Post("/", PostHandler)
	r.Post("/api/shorten", PostShortenHandler)
	r.Get("/{id}", GetHandler)
	return r
}
