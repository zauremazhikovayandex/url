package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// простой мок, чтобы не ходить в БД
type noopService struct{}

func (noopService) GetOriginalURL(_ context.Context, _ string) (string, error) { return "", nil }
func (noopService) GetURLsByUserID(_ context.Context, _ string) ([]postgres.URL, error) {
	return nil, nil
}
func (noopService) GetShortIDByOriginalURL(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (noopService) SaveURL(_ context.Context, _, _ string, _ string) error    { return nil }
func (noopService) DeleteForUser(_ context.Context, _ string, _ string) error { return nil }
func (noopService) BatchDelete(_ context.Context, _ []string, _ string) error { return nil }

func BenchmarkPostShortenJSON(b *testing.B) {
	config.InitConfig()
	config.AppConfig.StorageType = "File"

	s := httptest.NewServer(InitHandlers(noopService{}))
	defer s.Close()

	type req struct {
		URL string `json:"url"`
	}
	payload, _ := json.Marshal(req{URL: "https://example.com/bench"})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(s.URL+"/api/shorten", "application/json", bytes.NewReader(payload))
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
