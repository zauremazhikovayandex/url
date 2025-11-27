package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/auth"
	"github.com/zauremazhikovayandex/url/internal/gzip"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"github.com/zauremazhikovayandex/url/internal/services"
)

// URLPair - описывает пару короткой и оригинальной ссылок в ответах API
type URLPair struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// PostHandler - принимает URL в теле запроса (text/plain), сохраняет и возвращает короткую ссылку
func (h *Handler) PostHandler(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	timeStart := time.Now()

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "r.Body", "POST", http.StatusBadRequest, "Invalid request body")
		return
	}

	originalURL := strings.TrimSpace(string(body))

	shortURL, conflict, err := h.shortener.CreateShortURL(r.Context(), originalURL, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidURL):
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusBadRequest, "Invalid URL format")
		default:
			logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Storage ERROR: %s", err)})
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	status := http.StatusCreated
	if conflict {
		status = http.StatusConflict
	}

	// Ответ в text/plain
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	if shortURL != "" {
		if _, err := w.Write([]byte(shortURL)); err != nil {
			logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Write ERROR: %s", err)})
		}
	}
	logger.Logging.WriteToLog(timeStart, originalURL, "POST", status, shortURL)
}

// PostShortenHandler - принимает JSON {"url": "..."} и возвращает короткую ссылку в JSON
func (h *Handler) PostShortenHandler(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	timeStart := time.Now()

	type RequestPayload struct {
		URL string `json:"url"`
	}
	type ResponsePayload struct {
		Result string `json:"result"`
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid Content-Type")
		return
	}

	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.URL == "" {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	originalURL := strings.TrimSpace(payload.URL)
	shortURL, conflict, err := h.shortener.CreateShortURL(r.Context(), originalURL, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidURL):
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusBadRequest, "Invalid URL format")
		default:
			logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Storage ERROR: %s", err)})
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	status := http.StatusCreated
	if conflict {
		status = http.StatusConflict
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ResponsePayload{Result: shortURL})

	logger.Logging.WriteToLog(timeStart, originalURL, "POST", status, shortURL)
}

// PostShortenHandlerBatch - принимает массив JSON и возвращает массив коротких ссылок
func (h *Handler) PostShortenHandlerBatch(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	timeStart := time.Now()

	type BatchRequestItem struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}
	type BatchResponseItem struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid Content-Type")
		return
	}

	var requests []BatchRequestItem
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid JSON array")
		return
	}

	// Преобразуем в доменные BatchItem и отдаём сервису
	items := make([]services.BatchItem, 0, len(requests))
	for _, it := range requests {
		items = append(items, services.BatchItem{
			CorrelationID: it.CorrelationID,
			OriginalURL:   strings.TrimSpace(it.OriginalURL),
		})
	}

	results, err := h.shortener.CreateShortURLBatch(r.Context(), items, userID)
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Batch shorten ERROR: %s", err)})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var responses []BatchResponseItem
	for _, res := range results {
		responses = append(responses, BatchResponseItem{
			CorrelationID: res.CorrelationID,
			ShortURL:      res.ShortURL,
		})
		logger.Logging.WriteToLog(timeStart, res.ShortURL, "POST", http.StatusCreated, res.ShortURL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Failed to write batch response: %s", err)})
	}
}

// GetHandler - выполняет редирект 307 по id короткой ссылки
func (h *Handler) GetHandler(w http.ResponseWriter, r *http.Request) {
	timeStart := time.Now()
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "Missing ID")
		return
	}

	orig, deleted, err := h.shortener.Resolve(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, "URL not found", http.StatusBadRequest)
			logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "URL not found")
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusInternalServerError, "Resolve error")
		return
	}
	if deleted {
		http.Error(w, "URL deleted", http.StatusGone)
		logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusGone, "URL deleted")
		return
	}

	logger.Logging.WriteToLog(timeStart, orig, "GET", http.StatusTemporaryRedirect, id)
	http.Redirect(w, r, orig, http.StatusTemporaryRedirect)
}

// GetUserURLs - возвращает список ссылок пользователя
func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	urls, err := h.shortener.UserURLs(r.Context(), userID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(urls)
}

// GetDBPing - проверяет доступность подключения к БД
func (h *Handler) GetDBPing(w http.ResponseWriter, r *http.Request) {
	if err := h.shortener.PingDB(r.Context()); err != nil {
		http.Error(w, "fail DB connection", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteUserURLs - помечает на удаление список ссылок пользователя
func (h *Handler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil || len(ids) == 0 {
		http.Error(w, "Invalid JSON or empty ID list", http.StatusBadRequest)
		return
	}

	if err := h.shortener.DeleteUserURLs(r.Context(), ids, userID); err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Failed to mark URLs as deleted: %s", err)})
		http.Error(w, fmt.Sprintf("Failed to mark URLs as deleted: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// GzipMiddleware - возможность передавать данные в gzip
func (h *Handler) GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := gzip.NewCompressReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := gzip.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		next.ServeHTTP(ow, r)
	})
}
