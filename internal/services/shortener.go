package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	neturl "net/url"

	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
)

// Доменные ошибки, по ним HTTP/gRPC хендлеры мапят статусы.
var (
	ErrInvalidURL = errors.New("invalid url")
	ErrNotFound   = errors.New("url not found")
	ErrDeleted    = errors.New("url deleted")
	ErrForbidden  = errors.New("forbidden")
)

// Shortener — фасад над всей бизнес-логикой коротких ссылок.
type Shortener interface {
	CreateShortURL(ctx context.Context, originalURL, userID string) (shortURL string, conflict bool, err error)
	CreateShortURLBatch(ctx context.Context, items []BatchItem, userID string) ([]BatchResult, error)
	Resolve(ctx context.Context, id string) (originalURL string, deleted bool, err error)
	UserURLs(ctx context.Context, userID string) ([]URLPair, error)
	DeleteUserURLs(ctx context.Context, ids []string, userID string) error
	PingDB(ctx context.Context) error
	Stats(ctx context.Context, clientIP net.IP) (urls, users int, err error)
}

// URLPair — доменная модель пары ссылок для сервисного слоя.
type URLPair struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// BatchItem/BatchResult — доменные типы для batch-сокращения.
type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

type BatchResult struct {
	CorrelationID string
	ShortURL      string
}

// shortener — реализация фасада.
type shortener struct {
	repo        URLService
	baseURL     string
	storageType string
	store       *storage.Storage
	trustedNet  *net.IPNet
}

// NewShortener конструирует сервис, получая все зависимости извне.
func NewShortener(
	repo URLService,
	baseURL string,
	storageType string,
	store *storage.Storage,
	trustedNet *net.IPNet,
) Shortener {
	return &shortener{
		repo:        repo,
		baseURL:     baseURL,
		storageType: storageType,
		store:       store,
		trustedNet:  trustedNet,
	}
}

// ---- helpers ----

func isValidURL(rawURL string) bool {
	parsed, err := neturl.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func generateShortID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

func (s *shortener) makeShortURL(id string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, id)
}

// ---- реализация фасада ----

func (s *shortener) CreateShortURL(ctx context.Context, originalURL, userID string) (string, bool, error) {
	if !isValidURL(originalURL) {
		return "", false, ErrInvalidURL
	}

	id, err := generateShortID(8)
	if err != nil {
		return "", false, err
	}

	if s.storageType == "DB" {
		if err := s.repo.SaveURL(ctx, id, originalURL, userID); err != nil {
			// duplicate URL
			if errors.Is(err, postgres.ErrDuplicateOriginalURL) {
				existingID, getErr := s.repo.GetShortIDByOriginalURL(ctx, originalURL)
				if getErr != nil || existingID == "" {
					// конфликт, но без готового shortURL
					return "", true, nil
				}
				return s.makeShortURL(existingID), true, nil
			}
			return "", false, err
		}
	} else {
		if s.store == nil {
			return "", false, errors.New("no storage configured")
		}
		s.store.Set(id, originalURL)
	}

	return s.makeShortURL(id), false, nil
}

func (s *shortener) CreateShortURLBatch(ctx context.Context, items []BatchItem, userID string) ([]BatchResult, error) {
	var out []BatchResult

	for _, it := range items {
		u := it.OriginalURL
		if !isValidURL(u) {
			continue
		}
		id, err := generateShortID(8)
		if err != nil {
			continue
		}
		if s.storageType == "DB" {
			if err := s.repo.SaveURL(ctx, id, u, userID); err != nil {
				continue
			}
		} else {
			if s.store == nil {
				continue
			}
			s.store.Set(id, u)
		}
		out = append(out, BatchResult{
			CorrelationID: it.CorrelationID,
			ShortURL:      s.makeShortURL(id),
		})
	}

	return out, nil
}

func (s *shortener) Resolve(ctx context.Context, id string) (string, bool, error) {
	if id == "" {
		return "", false, ErrNotFound
	}

	if s.storageType == "DB" {
		orig, err := s.repo.GetOriginalURL(ctx, id)
		if err != nil {
			if errors.Is(err, postgres.ErrURLDeleted) {
				return "", true, nil
			}
			return "", false, ErrNotFound
		}
		return orig, false, nil
	}

	if s.store == nil {
		return "", false, ErrNotFound
	}
	orig, ok := s.store.Get(id)
	if !ok || orig == "" {
		return "", false, ErrNotFound
	}
	return orig, false, nil
}

func (s *shortener) UserURLs(ctx context.Context, userID string) ([]URLPair, error) {
	urls, err := s.repo.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, nil
	}

	out := make([]URLPair, 0, len(urls))
	for _, u := range urls {
		if u.Deleted == 1 {
			continue
		}
		out = append(out, URLPair{
			ShortURL:    s.makeShortURL(u.ID),
			OriginalURL: u.OriginalURL,
		})
	}
	return out, nil
}

func (s *shortener) DeleteUserURLs(ctx context.Context, ids []string, userID string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.BatchDelete(ctx, ids, userID)
}

func (s *shortener) PingDB(ctx context.Context) error {
	conn, err := postgres.SQLInstance()
	if conn == nil || err != nil {
		return err
	}
	return conn.Ping()
}

func (s *shortener) Stats(ctx context.Context, clientIP net.IP) (int, int, error) {
	if s.trustedNet == nil || clientIP == nil || !s.trustedNet.Contains(clientIP) {
		return 0, 0, ErrForbidden
	}

	if s.storageType == "DB" {
		u, us, err := postgres.CountStats(ctx)
		if err != nil {
			return 0, 0, err
		}
		return u, us, nil
	}

	if s.store == nil {
		return 0, 0, errors.New("no storage")
	}

	return s.store.Len(), 0, nil
}
