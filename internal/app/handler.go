package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/gzip"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func generateShortID(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

func isValidURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func PostHandler(w http.ResponseWriter, r *http.Request) {

	timeStart := time.Now()
	storageType := config.AppConfig.StorageType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "r.Body", "POST", http.StatusBadRequest, "Invalid request body")
		return
	}

	originalURL := strings.TrimSpace(string(body))

	if !isValidURL(originalURL) {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusBadRequest, "Invalid URL format")
		return
	}

	id, err := generateShortID(8)
	if err != nil || id == "" {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusInternalServerError, "Internal server error")
		return
	}

	if storageType == "DB" {
		err = postgres.InsertURL(ctx, id, originalURL)
		if err != nil {
			logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Storage ERROR: %s", err)})
		}
	} else {
		storage.Store.Set(id, originalURL)
	}

	shortURL := fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(shortURL))
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Storage ERROR: %s", err)})
	}
	logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusCreated, shortURL)

}

func PostShortenHandler(w http.ResponseWriter, r *http.Request) {

	timeStart := time.Now()
	storageType := config.AppConfig.StorageType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Структура для чтения входного JSON
	type RequestPayload struct {
		URL string `json:"url"`
	}

	// Структура для ответа
	type ResponsePayload struct {
		Result string `json:"result"`
	}

	var payload RequestPayload

	// Проверка Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid Content-Type")
		return
	}

	// Декодирование JSON-запроса
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.URL == "" {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	originalURL := strings.TrimSpace(payload.URL)
	if !isValidURL(originalURL) {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusBadRequest, "Invalid URL format")
		return
	}

	id, err := generateShortID(8)
	if err != nil || id == "" {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusInternalServerError, "Failed to generate short ID")
		return
	}

	if storageType == "DB" {
		err = postgres.InsertURL(ctx, id, originalURL)
		if err != nil {
			logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Storage ERROR: %s", err)})
		}
	} else {
		storage.Store.Set(id, originalURL)
	}

	shortURL := fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id)

	// Отправка JSON-ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ResponsePayload{Result: shortURL})

	logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusCreated, shortURL)
}

func PostShortenHandlerBatch(w http.ResponseWriter, r *http.Request) {
	timeStart := time.Now()
	storageType := config.AppConfig.StorageType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Структуры запроса и ответа
	type BatchRequestItem struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}

	type BatchResponseItem struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}

	var requests []BatchRequestItem

	// Проверка Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid Content-Type")
		return
	}

	// Чтение и декодирование JSON-массива
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "POST", http.StatusBadRequest, "Invalid JSON array")
		return
	}

	var responses []BatchResponseItem

	for _, item := range requests {
		originalURL := strings.TrimSpace(item.OriginalURL)

		if !isValidURL(originalURL) {
			// Пропускаем или логируем ошибочный элемент (можно изменить поведение при необходимости)
			logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusBadRequest, fmt.Sprintf("Invalid URL format for correlation_id=%s", item.CorrelationID))
			continue
		}

		id, err := generateShortID(8)
		if err != nil || id == "" {
			logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusInternalServerError, fmt.Sprintf("Failed to generate ID for correlation_id=%s", item.CorrelationID))
			continue
		}

		if storageType == "DB" {
			err = postgres.InsertURL(ctx, id, originalURL)
			if err != nil {
				logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Storage ERROR for correlation_id=%s: %s", item.CorrelationID, err)})
				continue
			}
		} else {
			storage.Store.Set(id, originalURL)
		}

		shortURL := fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id)

		responses = append(responses, BatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		})
		logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusCreated, shortURL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("Failed to write batch response: %s", err)})
	}
}

func GetHandler(w http.ResponseWriter, r *http.Request) {

	timeStart := time.Now()
	storageType := config.AppConfig.StorageType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "Missing ID")
		return
	}

	if storageType == "DB" {
		originalURL, err := postgres.SelectURL(ctx, id)
		if err != nil || originalURL == "" {
			http.Error(w, "URL not found", http.StatusBadRequest)
			logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "URL not found")
			return
		}
		logger.Logging.WriteToLog(timeStart, originalURL, "GET", http.StatusTemporaryRedirect, id)
		http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)

	} else {
		originalURL, ok := storage.Store.Get(id)
		if !ok || originalURL == "" {
			http.Error(w, "URL not found", http.StatusBadRequest)
			logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "URL not found")
			return
		}
		logger.Logging.WriteToLog(timeStart, originalURL, "GET", http.StatusTemporaryRedirect, id)
		http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
	}

}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		// Декодирование входящего тела, если оно сжато
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := gzip.NewCompressReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		// Сжатие ответа, если клиент его поддерживает
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := gzip.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		next.ServeHTTP(ow, r)
	})
}

func GetDBPing(w http.ResponseWriter, r *http.Request) {
	conn, err := postgres.SQLInstance()
	if conn == nil || err != nil {
		http.Error(w, "fail DB connection", http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
