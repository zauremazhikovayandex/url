package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/services"
)

type noopShortener struct{}

var _ services.Shortener = (*noopShortener)(nil)

func (noopShortener) CreateShortURL(ctx context.Context, originalURL, userID string) (string, bool, error) {
	return "", false, nil
}

func (noopShortener) CreateShortURLBatch(ctx context.Context, items []services.BatchItem, userID string) ([]services.BatchResult, error) {
	return nil, nil
}

func (noopShortener) Resolve(ctx context.Context, id string) (string, bool, error) {
	return "", false, nil
}

func (noopShortener) UserURLs(ctx context.Context, userID string) ([]services.URLPair, error) {
	return nil, nil
}

func (noopShortener) DeleteUserURLs(ctx context.Context, ids []string, userID string) error {
	return nil
}

func (noopShortener) PingDB(ctx context.Context) error {
	return nil
}

func (noopShortener) Stats(ctx context.Context, clientIP net.IP) (int, int, error) {
	return 0, 0, nil
}

func BenchmarkPostShortenJSON(b *testing.B) {
	config.InitConfig()
	logger.New("info")

	srv := httptest.NewServer(InitHandlers(noopShortener{}))
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
