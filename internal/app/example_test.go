// example_test.go
package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
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

func setupMemoryApp() *Handler {
	config.AppConfig = &config.Config{
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

	// Глушим весь лог в примерах:
	logger.Log = discardDriver{}        // ничего не пишет
	logger.Logging = noopAccessLogger{} // и access-лог тоже молчит

	storage.InitStorage()
	return &Handler{}
}

// routerForGet создает chi.Router только для примера с GET /{id},
// чтобы chi проставил URLParam("id").
func routerForGet(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/{id}", h.GetHandler)
	return r
}

// ExampleHandler_PostHandler демонстрирует создание короткой ссылки через POST-тело (text/plain).
func ExampleHandler_PostHandler() {
	h := setupMemoryApp()

	body := "https://example.com/hello"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	// Явно указывать Content-Type не обязательно для этого хендлера, он читает «как есть».
	w := httptest.NewRecorder()

	h.PostHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status) // ожидается 201 Created
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))
	// Тело ответа — полный короткий URL, например: http://localhost:8080/AbCdEf12
	// Оно содержит случайный id, поэтому в примере мы не фиксируем его значение.
	// Output:
	// (пусто — пример демонстрационный, без assert-а по выводу)
}

// ExampleHandler_PostShortenHandler демонстрирует создание короткой ссылки через JSON (application/json).
func ExampleHandler_PostShortenHandler() {
	h := setupMemoryApp()

	payload := map[string]string{"url": "https://golang.org"}
	b, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.PostShortenHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status) // ожидается 201 Created
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))
	// Тело — {"result": "<short_url>"} со случайным id. В примере не проверяем детально.
	// Output:
	// (пусто — демонстрационный пример)
}

// ExampleHandler_PostShortenHandler_invalid демонстрирует валидационную ошибку на неверный URL с JSON.
func ExampleHandler_PostShortenHandler_invalid() {
	h := setupMemoryApp()

	payload := map[string]string{"url": "not a url"}
	b, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.PostShortenHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	fmt.Print(resp.Status, "\n")
	fmt.Print(buf.String())
	// Output:
	// 400 Bad Request
	// Invalid URL format
}

// ExampleHandler_PostShortenHandlerBatch демонстрирует пакетное создание ссылок.
func ExampleHandler_PostShortenHandlerBatch() {
	h := setupMemoryApp()

	batchReq := []map[string]string{
		{"correlation_id": "1", "original_url": "https://a.example"},
		{"correlation_id": "2", "original_url": "https://b.example"},
	}
	b, _ := json.Marshal(batchReq)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.PostShortenHandlerBatch(w, req)
	_ = w.Result().Body.Close()
}

// ExampleHandler_GetHandler демонстрирует редирект 307 по существующему id.
func ExampleHandler_GetHandler() {
	h := setupMemoryApp()

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

// ExampleHandler_GzipMiddleware демонстрирует прием gzip-сжатого тела для POST-хендлера.
func ExampleHandler_GzipMiddleware() {
	h := setupMemoryApp()

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
	_ = w.Result().Body.Close()
}

// ExampleHandler_GetDBPing демонстрирует проверку пинга БД.
// В Memory-конфигурации подключения к БД нет — ожидаем 500.
func ExampleHandler_GetDBPing() {
	h := setupMemoryApp()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.GetDBPing(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println(resp.Status) // ожидается 500 Internal Server Error
	// Output:
	// 500 Internal Server Error
}
