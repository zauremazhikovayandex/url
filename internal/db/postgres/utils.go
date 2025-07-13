package postgres

import (
	"context"
)

func SelectURL(ctx context.Context, id string) (string, error) {
	instance, err := SQLInstance()
	if err != nil {
		return "", err
	}
	db := instance.PgSQL

	timeoutCtx, cancel := context.WithTimeout(ctx, instance.Timeout)
	defer cancel()

	query := "SELECT \"originalURL\" FROM urls WHERE \"id\" = $1"

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

	query := "INSERT INTO urls (\"id\", \"originalURL\") VALUES ($1, $2)"

	_, err = db.Exec(timeoutCtx, query, id, originalURL)
	return err
}

func CreateTables() error {
	db, err := SQLInstance()
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = db.PgSQL.Query(ctx,
		`CREATE TABLE IF NOT EXISTS urls (
        "id" TEXT,
        "originalURL" TEXT
      )`)
	if err != nil {
		return err
	}
	return nil
}

func PrepareDB() error {

	err := CreateTables()
	return err

}
