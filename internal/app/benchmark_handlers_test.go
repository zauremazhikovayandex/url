package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/services"
)

// --- мок URLService, чтоб не ходить в БД ---
type noopService struct{}

var _ services.URLService = (*noopService)(nil)

func (noopService) GetOriginalURL(context.Context, string) (string, error)          { return "", nil }
func (noopService) GetURLsByUserID(context.Context, string) ([]postgres.URL, error) { return nil, nil }
func (noopService) GetShortIDByOriginalURL(context.Context, string) (string, error) { return "", nil }
func (noopService) SaveURL(context.Context, string, string, string) error           { return nil }
func (noopService) DeleteForUser(context.Context, string, string) error             { return nil }
func (noopService) BatchDelete(context.Context, []string, string) error             { return nil }

// --- Бенч: POST /api/shorten ---
func BenchmarkPostShortenJSON(b *testing.B) {
	config.InitConfig()
	config.AppConfig.StorageType = "DB"

	srv := httptest.NewServer(InitHandlers(noopService{}))
	defer srv.Close()

	type req struct {
		URL string `json:"url"`
	}
	payload, _ := json.Marshal(req{URL: "https://example.com/bench"})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := http.Post(srv.URL+"/api/shorten", "application/json", bytes.NewReader(payload))
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			b.Fatalf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
