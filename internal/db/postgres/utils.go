// Package postgres - Пакет по работе с БД Postgres
package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"log"
	"strings"
)

// URL представляет запись о короткой ссылке в БД.
type URL struct {
	ID          string
	OriginalURL string
	Deleted     int
}

// ErrURLDeleted сигнализирует, что ссылка помечена как удаленная.
var ErrURLDeleted = errors.New("url_deleted")

// ErrDuplicateOriginalURL сигнализирует, что оригинальный URL уже существует.
var ErrDuplicateOriginalURL = errors.New("duplicate_original_url")

// SelectURL возвращает оригинальный URL по id.
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

// InsertURL сохраняет новый URL, возвращая ошибку при дубликате.
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
		return ErrDuplicateOriginalURL
	}
	if err != nil {
		return err
	}
	return nil
}

// SelectIDByOriginalURL возвращает id по оригинальному URL.
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

// SelectURLsByUser возвращает все активные URL пользователя.
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

// CreateTables создает необходимые таблицы, если их нет.
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

// PrepareDB выполняет начальную подготовку БД (миграции/создание таблиц).
func PrepareDB(db *SQLConnection) {
	err := CreateTables(db)
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("DB CreateTables ERROR: %s", err)})
	}
}

// DeleteURL помечает ссылку как удаленную по id и userID.
func DeleteURL(ctx context.Context, id string, userID string) error {
	if id == "" {
		return nil
	}

	instance, err := SQLInstance()
	if err != nil {
		return err
	}
	db := instance.PgSQL

	args := []interface{}{userID, id}

	query := "UPDATE urls SET deleted = 1 WHERE userID = $1 AND id = $2"

	ctxWithTimeout, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	_, err = db.Exec(ctxWithTimeout, query, args...)
	if err != nil {
		log.Printf("Batch delete failed: %v", err)
	}
	return err
}

// BatchDeleteURLs помечает на удаление список ссылок по id и userID.
func BatchDeleteURLs(ctx context.Context, ids []string, userID string) error {
	if len(ids) == 0 {
		return nil
	}

	instance, err := SQLInstance()
	if err != nil {
		return err
	}
	db := instance.PgSQL

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, userID)

	placeholders := make([]string, 0, len(ids))
	for i, id := range ids {
		args = append(args, id)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+2))
	}

	query := fmt.Sprintf(`UPDATE urls SET deleted = 1 WHERE userID = $1 AND id IN (%s)`, strings.Join(placeholders, ", "))

	ctxWithTimeout, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	_, err = db.Exec(ctxWithTimeout, query, args...)
	return err
}

// CountStats возвращает количество активных URL и количество пользователей.
func CountStats(ctx context.Context) (int, int, error) {
	instance, err := SQLInstance()
	if err != nil {
		return 0, 0, err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	var urls, users int
	// одной выборкой
	row := db.QueryRow(timeoutCtx, `
		SELECT 
		  COUNT(*) FILTER (WHERE deleted = 0) AS urls,
		  COUNT(DISTINCT CASE WHEN deleted = 0 THEN userID END) AS users
		FROM urls`)
	if err := row.Scan(&urls, &users); err != nil {
		return 0, 0, err
	}
	return urls, users, nil
}
