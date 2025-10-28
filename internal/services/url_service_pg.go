// Package services содержит бизнес-логику поверх слоев хранилища.
package services

import (
	"context"
	"errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
)

// PostgresURLService реализует операции с URL поверх PostgreSQL.
type PostgresURLService struct{}

// GetOriginalURL возвращает оригинальный URL по id.
func (s *PostgresURLService) GetOriginalURL(ctx context.Context, id string) (string, error) {
	return postgres.SelectURL(ctx, id)
}

// GetURLsByUserID возвращает список ссылок пользователя.
func (s *PostgresURLService) GetURLsByUserID(ctx context.Context, userID string) ([]postgres.URL, error) {
	return postgres.SelectURLsByUser(ctx, userID)
}

// GetShortIDByOriginalURL возвращает id по оригинальному URL.
func (s *PostgresURLService) GetShortIDByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	return postgres.SelectIDByOriginalURL(ctx, originalURL)
}

// SaveURL сохраняет новую короткую ссылку.
func (s *PostgresURLService) SaveURL(ctx context.Context, id string, originalURL string, userID string) error {
	err := postgres.InsertURL(ctx, id, originalURL, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return postgres.ErrDuplicateOriginalURL
		}
		return err
	}
	return nil
}

// DeleteForUser помечает ссылку как удаленную для пользователя.
func (s *PostgresURLService) DeleteForUser(ctx context.Context, id string, userID string) error {
	return postgres.DeleteURL(ctx, id, userID)
}

// BatchDelete помечает на удаление набор ссылок пользователя.
func (s *PostgresURLService) BatchDelete(ctx context.Context, ids []string, userID string) error {
	return postgres.BatchDeleteURLs(ctx, ids, userID)
}
