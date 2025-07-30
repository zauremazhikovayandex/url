package services

import (
	"context"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
)

type URLService interface {
	GetOriginalURL(ctx context.Context, id string) (string, error)
	GetURLsByUserID(ctx context.Context, userID string) ([]postgres.URL, error)
	GetShortIDByOriginalURL(ctx context.Context, originalURL string) (string, error)
	SaveURL(ctx context.Context, id string, originalURL string, userID string) error
	DeleteForUser(ctx context.Context, id string, userID string) error
}
