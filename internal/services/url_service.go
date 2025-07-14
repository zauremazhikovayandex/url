package services

import "context"

type URLService interface {
	GetOriginalURL(ctx context.Context, id string) (string, error)
	GetShortIDByOriginalURL(ctx context.Context, originalURL string) (string, error)
	SaveURL(ctx context.Context, id, originalURL string) error
}
