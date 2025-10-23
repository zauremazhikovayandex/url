package app

import (
	"net/http"
	"strings"

	"github.com/zauremazhikovayandex/url/internal/auth"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/services"
)

type Handler struct {
	urlService services.URLService
}

// Chain применяет мидлвары в заданном порядке (первый — самый внешний, как в chi.Use)
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func (h *Handler) methodOnly(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

func (h *Handler) methods(allowed []string, next http.HandlerFunc) http.HandlerFunc {
	allowSet := make(map[string]struct{}, len(allowed))
	for _, m := range allowed {
		allowSet[m] = struct{}{}
	}
	allowHeader := strings.Join(allowed, ", ")

	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := allowSet[r.Method]; !ok {
			w.Header().Set("Allow", allowHeader)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

func (h *Handler) fallbackRootOrID(w http.ResponseWriter, r *http.Request) {
	// POST "/" -> h.PostHandler
	if r.URL.Path == "/" {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.PostHandler(w, r)
		return
	}

	// Иначе это GET "/{id}"
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлечь id из первого сегмента
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" || strings.HasPrefix(id, "api/") {
		http.NotFound(w, r)
		return
	}

	h.GetHandler(w, r)
}

func InitHandlers(urlService services.URLService) http.Handler {
	h := &Handler{urlService: urlService}

	mux := http.NewServeMux()

	// /api/shorten (POST)
	mux.HandleFunc("/api/shorten", h.methodOnly(http.MethodPost, h.PostShortenHandler))

	// /api/shorten/batch (POST)
	mux.HandleFunc("/api/shorten/batch", h.methodOnly(http.MethodPost, h.PostShortenHandlerBatch))

	// /api/user/urls (GET, DELETE)
	mux.HandleFunc("/api/user/urls", h.methods([]string{http.MethodGet, http.MethodDelete}, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetUserURLs(w, r)
		case http.MethodDelete:
			h.DeleteUserURLs(w, r)
		}
	}))

	// /ping (GET)
	mux.HandleFunc("/ping", h.methodOnly(http.MethodGet, h.GetDBPing))

	// fallback для "/" и "/{id}"
	mux.HandleFunc("/", h.fallbackRootOrID)

	return Chain(mux,
		auth.Middleware,
		h.GzipMiddleware,
		logger.RequestLogger,
	)
}
