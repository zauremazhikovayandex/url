package services

import (
	"context"
	"errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
)

type PostgresURLService struct{}

func (s *PostgresURLService) GetOriginalURL(ctx context.Context, id string) (string, error) {
	return postgres.SelectURL(ctx, id)
}

func (s *PostgresURLService) GetURLsByUserID(ctx context.Context, userID string) ([]postgres.URL, error) {
	return postgres.SelectURLsByUser(ctx, userID)
}

func (s *PostgresURLService) GetShortIDByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	return postgres.SelectIDByOriginalURL(ctx, originalURL)
}

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

func (s *PostgresURLService) DeleteForUser(ctx context.Context, id string, userID string) error {
	return postgres.DeleteURL(ctx, id, userID)
}

func (s *PostgresURLService) BatchDelete(ctx context.Context, ids []string, userID string) error {
	return postgres.BatchDeleteURLs(ctx, ids, userID)
}
