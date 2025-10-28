package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
)

type discardDriver struct{}

func (discardDriver) Debug(*message.LogMessage) {}
func (discardDriver) Info(*message.LogMessage)  {}
func (discardDriver) Warn(*message.LogMessage)  {}
func (discardDriver) Error(*message.LogMessage) {}
func (discardDriver) Fatal(*message.LogMessage) {}
func (discardDriver) Panic(*message.LogMessage) {}

type noopAccessLogger struct{}

func (noopAccessLogger) WriteToLog(time.Time, string, string, int, string) {}

// setupMemoryApp настраивает окружение и возвращает teardown для безопасного восстановления глобалок.
func setupMemoryApp() (*Handler, func()) {
	// save globals
	prevCfg := config.AppConfig
	prevStore := storage.Store
	prevLog := logger.Log
	prevLogging := logger.Logging

	// local, isolated config
	cfg := &config.Config{
		ServerAddr:     ":8080",
		BaseURL:        "http://localhost:8080",
		UseFileStorage: "",
		FileStorage:    "",
		PGConfig:       &config.PostgresConfig{DBConnection: "", DBTimeout: 10},
		StorageType:    "Memory",
		JWTSecretKey:   "example",
		JWTTokenExp:    0,
		JWTCookieName:  "auth_token",
	}
	config.AppConfig = cfg

	// mute logs
	logger.Log = discardDriver{}
	logger.Logging = noopAccessLogger{}

	// fresh in-mem store
	storage.Store = &storage.Storage{}
	storage.InitStorage()

	teardown := func() {
		config.AppConfig = prevCfg
		storage.Store = prevStore
		logger.Log = prevLog
		logger.Logging = prevLogging
	}

	return &Handler{}, teardown
}

func routerForGet(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/{id}", h.GetHandler)
	return r
}

// ExampleHandler_PostHandler — создание короткой ссылки (text/plain).
func ExampleHandler_PostHandler() {
	h, done := setupMemoryApp()
	defer done()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/hello"))
	w := httptest.NewRecorder()
	h.PostHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.Status)
}

// ExampleHandler_PostShortenHandler — создание через JSON.
func ExampleHandler_PostShortenHandler() {
	h, done := setupMemoryApp()
	defer done()

	b, _ := json.Marshal(map[string]string{"url": "https://golang.org"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.PostShortenHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.Status)
	fmt.Println(resp.Header.Get("Content-Type"))
}

// ExampleHandler_PostShortenHandler_invalid — ошибка валидации (JSON с не-URL).
func ExampleHandler_PostShortenHandler_invalid() {
	h, done := setupMemoryApp()
	defer done()

	b, _ := json.Marshal(map[string]string{"url": "not a url"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.PostShortenHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println(resp.Status)
	fmt.Print(string(body))
}

// ExampleHandler_PostShortenHandlerBatch — пакетное создание: проверяем статус и размер ответа.
func ExampleHandler_PostShortenHandlerBatch() {
	h, done := setupMemoryApp()
	defer done()

	batchReq := []map[string]string{
		{"correlation_id": "1", "original_url": "https://a.example"},
		{"correlation_id": "2", "original_url": "https://b.example"},
	}
	b, _ := json.Marshal(batchReq)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.PostShortenHandlerBatch(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	var out []map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&out)

	fmt.Println(resp.Status)
	fmt.Println(len(out))
}

// ExampleHandler_GetHandler — редирект по фиксированному id.
func ExampleHandler_GetHandler() {
	h, done := setupMemoryApp()
	defer done()

	id := "fixedID1"
	original := "https://example.com/landing"
	storage.Store.Set(id, original)

	r := routerForGet(h)
	req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.Status)
	fmt.Println(resp.Header.Get("Location"))
}

// ExampleHandler_GzipMiddleware — входное тело gzipped, проверяем статус.
func ExampleHandler_GzipMiddleware() {
	h, done := setupMemoryApp()
	defer done()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("https://example.org/path"))
	_ = zw.Close()

	handler := h.GzipMiddleware(http.HandlerFunc(h.PostHandler))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.Status)
}
