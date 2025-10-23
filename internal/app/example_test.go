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
	"net/http"
	"net/http/httptest"
	"strings"
)

// setupMemoryApp минимально конфигурирует приложение для примеров: память + базовый URL.
// БД не используется, поэтому urlService может быть nil.
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
	storage.InitStorage()
	return &Handler{} // urlService не нужен для Memory-пути
}

// routerForGet создает chi.Router только для примера с GET /{id},
// чтобы chi проставил URLParam("id").
func routerForGet(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/{id}", h.GetHandler)
	return r
}

// ExamplePostHandler демонстрирует создание короткой ссылки через POST-тело (text/plain).
func ExamplePostHandler() {
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

// ExamplePostShortenHandler демонстрирует создание короткой ссылки через JSON (application/json).
func ExamplePostShortenHandler() {
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

// ExamplePostShortenHandler_invalid демонстрирует валидационную ошибку на неверный URL с JSON.
func ExamplePostShortenHandler_invalid() {
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

// ExamplePostShortenHandlerBatch демонстрирует пакетное создание ссылок.
func ExamplePostShortenHandlerBatch() {
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

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status) // ожидается 201 Created
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))
	// Output:
	// (пусто — демонстрационный пример)
}

// ExampleGetHandler демонстрирует редирект 307 по существующему id.
func ExampleGetHandler() {
	h := setupMemoryApp()

	// Положим заранее в память "ручной" id, чтобы редирект был детерминированным.
	id := "fixedID1"
	original := "https://example.com/landing"
	storage.Store.Set(id, original)

	// chi нужен, чтобы передать {id} в URLParam.
	r := routerForGet(h)

	req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)                   // ожидается 307 Temporary Redirect
	fmt.Println("Location:", resp.Header.Get("Location")) // ожидается исходный URL
	// Output:
	// 307 Temporary Redirect
	// https://example.com/landing
}

// ExampleGzipMiddleware демонстрирует прием gzip-сжатого тела для POST-хендлера.
func ExampleGzipMiddleware() {
	h := setupMemoryApp()

	// Подготовим gzip-тело с корректным URL внутри.
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("https://example.org/path"))
	_ = zw.Close()

	// Оборачиваем PostHandler в GzipMiddleware.
	handler := h.GzipMiddleware(http.HandlerFunc(h.PostHandler))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip") // чтобы ответ тоже мог быть сжат (если поддерживается)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status) // ожидается 201 Created
	// Output:
	// (пусто — демонстрационный пример)
}

// ExampleGetDBPing демонстрирует проверку пинга БД.
// В Memory-конфигурации подключения к БД нет — ожидаем 500.
func ExampleGetDBPing() {
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
