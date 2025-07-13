package postgres

import (
	"context"
	"fmt"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
)

func SelectURL(ctx context.Context, id string) (string, error) {
	instance, err := SQLInstance()
	if err != nil {
		return "", err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "SELECT originalURL FROM urls WHERE id = $1"

	var originalURL string
	err = db.QueryRow(timeoutCtx, query, id).Scan(&originalURL)
	if err != nil {
		return "", err
	}

	return originalURL, nil
}

func InsertURL(ctx context.Context, id string, originalURL string) error {
	instance, err := SQLInstance()
	if err != nil {
		return err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "INSERT INTO urls (id, originalURL) VALUES ($1, $2)"

	_, err = db.Exec(timeoutCtx, query, id, originalURL)
	return err
}

func CreateTables(db *SQLConnection) error {
	ctx := context.Background()
	_, err := db.PgSQL.Query(ctx,
		`CREATE TABLE IF NOT EXISTS urls (
        id TEXT,
        originalURL TEXT
      )`)
	if err != nil {
		return err
	}
	return nil
}

func PrepareDB() {
	db, err := SQLInstance()
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("DB Prepare ERROR: %s", err)})
		return
	}
	err = CreateTables(db)
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("DB CreateTables ERROR: %s", err)})
	}
}
