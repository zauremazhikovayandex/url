package app

import (
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

	storage.Store.Set(id, originalURL)

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

	storage.Store.Set(id, originalURL)

	shortURL := fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id)

	// Отправка JSON-ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ResponsePayload{Result: shortURL})

	logger.Logging.WriteToLog(timeStart, originalURL, "POST", http.StatusCreated, shortURL)
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	timeStart := time.Now()
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "Missing ID")
		return
	}

	originalURL, ok := storage.Store.Get(id)
	if !ok || originalURL == "" {
		http.Error(w, "URL not found", http.StatusBadRequest)
		logger.Logging.WriteToLog(timeStart, "", "GET", http.StatusBadRequest, "URL not found")
		return
	}

	logger.Logging.WriteToLog(timeStart, originalURL, "GET", http.StatusTemporaryRedirect, id)
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
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
