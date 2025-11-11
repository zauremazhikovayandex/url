// Package services содержит бизнес-логику поверх слоев хранения данных.
package services

import (
	"context"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
)

// URLService описывает набор операций над короткими ссылками и пользовательскими данными.
// Конкретная реализация (например, работа через PostgreSQL) должна обеспечивать эти методы.
type URLService interface {
	// GetOriginalURL возвращает исходный URL по короткому идентификатору.
	GetOriginalURL(ctx context.Context, id string) (string, error)
	// GetURLsByUserID возвращает список активных ссылок пользователя.
	GetURLsByUserID(ctx context.Context, userID string) ([]postgres.URL, error)
	// GetShortIDByOriginalURL возвращает короткий идентификатор по исходному URL.
	GetShortIDByOriginalURL(ctx context.Context, originalURL string) (string, error)
	// SaveURL сохраняет новую короткую ссылку для пользователя.
	SaveURL(ctx context.Context, id string, originalURL string, userID string) error
	// DeleteForUser помечает ссылку как удаленную для указанного пользователя.
	DeleteForUser(ctx context.Context, id string, userID string) error
	// BatchDelete помечает на удаление набор ссылок пользователя.
	BatchDelete(ctx context.Context, ids []string, userID string) error
	// GetStats агрегированная статистика
	GetStats(ctx context.Context) (urls int, users int, err error)
}
