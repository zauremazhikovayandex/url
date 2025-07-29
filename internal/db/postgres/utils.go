package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"strings"
)

type URL struct {
	ID          string
	OriginalURL string
	Deleted     int
}

var ErrURLDeleted = errors.New("url_deleted")

func SelectURL(ctx context.Context, id string) (string, error) {
	instance, err := SQLInstance()
	if err != nil {
		return "", err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "SELECT id, originalURL, deleted FROM urls WHERE id = $1"

	var u URL
	err = db.QueryRow(timeoutCtx, query, id).Scan(&u.ID, &u.OriginalURL, &u.Deleted)
	if err != nil {
		return "", err
	}
	if u.Deleted == 1 {
		return u.OriginalURL, ErrURLDeleted
	}

	return u.OriginalURL, nil
}

func InsertURL(ctx context.Context, id string, originalURL string, userID string) error {
	instance, err := SQLInstance()
	if err != nil {
		return err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "INSERT INTO urls (id, originalURL, userID) VALUES ($1, $2, $3) ON CONFLICT (originalURL) DO NOTHING RETURNING id;"

	var returnedID string
	err = db.QueryRow(timeoutCtx, query, id, originalURL, userID).Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("duplicate_original_url")
	}
	if err != nil {
		return err
	}
	return nil
}

func SelectIDByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	instance, err := SQLInstance()
	if err != nil {
		return "", err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "SELECT id FROM urls WHERE originalURL = $1"

	var id string
	err = db.QueryRow(timeoutCtx, query, originalURL).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func SelectURLsByUser(ctx context.Context, userID string) ([]URL, error) {
	instance, err := SQLInstance()
	if err != nil {
		return nil, err
	}
	db := instance.PgSQL
	ctx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "SELECT id, originalURL, deleted FROM urls WHERE userID = $1"
	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []URL
	for rows.Next() {
		var u URL
		if err := rows.Scan(&u.ID, &u.OriginalURL, &u.Deleted); err != nil {
			return nil, err
		}
		if u.Deleted == 0 {
			results = append(results, u)
		}
	}
	return results, nil
}

func CreateTables(db *SQLConnection) error {
	ctx := context.Background()
	_, err := db.PgSQL.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS urls (
			id TEXT,
			userID TEXT,
			originalURL TEXT UNIQUE,
			deleted INTEGER DEFAULT 0
		)`)
	if err != nil {
		return err
	}
	return nil
}

func PrepareDB(db *SQLConnection) {
	err := CreateTables(db)
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("DB CreateTables ERROR: %s", err)})
	}
}

func BatchDeleteURLs(ctx context.Context, ids []string, userID string) error {
	if len(ids) == 0 {
		return nil
	}

	instance, err := SQLInstance()
	if err != nil {
		return err
	}
	db := instance.PgSQL

	args := []interface{}{userID}
	params := make([]string, len(ids))
	for i, id := range ids {
		args = append(args, id)
		params[i] = fmt.Sprintf("$%d", i+2)
	}

	query := fmt.Sprintf(`
		UPDATE urls SET deleted = 1
		WHERE userID = $1 AND id IN (%s)
	`, strings.Join(params, ", "))

	ctxWithTimeout, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	_, err = db.Exec(ctxWithTimeout, query, args...)
	return err
}
