package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/services"
)

type noopURLService struct{}

var _ services.URLService = (*noopURLService)(nil)

func (n *noopURLService) GetOriginalURL(_ context.Context, _ string) (string, error) { return "", nil }
func (n *noopURLService) GetURLsByUserID(_ context.Context, _ string) ([]postgres.URL, error) {
	return nil, nil
}
func (n *noopURLService) GetShortIDByOriginalURL(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (n *noopURLService) SaveURL(_ context.Context, _, _ string, _ string) error    { return nil }
func (n *noopURLService) DeleteForUser(_ context.Context, _ string, _ string) error { return nil }
func (n *noopURLService) BatchDelete(_ context.Context, _ []string, _ string) error { return nil }

// --- бенчмарк POST /api/shorten ---

func BenchmarkPostShortenJSON(b *testing.B) {
	config.InitConfig()
	config.AppConfig.StorageType = "File"

	srv := httptest.NewServer(InitHandlers(&noopURLService{}))
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
			resp.Body.Close()
			b.Fatalf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
