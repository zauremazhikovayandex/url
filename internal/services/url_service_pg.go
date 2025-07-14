package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
)

type PostgresURLService struct{}

func (s *PostgresURLService) GetOriginalURL(ctx context.Context, id string) (string, error) {
	return postgres.SelectURL(ctx, id)
}

func (s *PostgresURLService) GetShortIDByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	return postgres.SelectIDByOriginalURL(ctx, originalURL)
}

func (s *PostgresURLService) SaveURL(ctx context.Context, id, originalURL string) error {
	err := postgres.InsertURL(ctx, id, originalURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("duplicate_original_url")
		}
		return err
	}
	return nil
}
