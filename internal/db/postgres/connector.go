// Package postgres содержит доступ к БД PostgreSQL и вспомогательные функции.
package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"time"
)

// SQLConnection инкапсулирует пул соединений и таймауты БД.
type SQLConnection struct {
	PgSQL   *pgxpool.Pool
	PgConn  *pgxpool.Conn
	Timeout time.Duration
}

var (
	pgSQL *SQLConnection
)

// SQLInstance лениво инициализирует и возвращает singleton-подключение к БД.
func SQLInstance() (*SQLConnection, error) {

	if pgSQL != nil {
		return pgSQL, nil
	}

	cfg := config.AppConfig.PGConfig
	timeout := time.Duration(cfg.DBTimeout) * time.Second

	pgCfg, err := pgxpool.ParseConfig(cfg.DBConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PG config: %w", err)
	}

	dbPool, err := pgxpool.ConnectConfig(context.Background(), pgCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PG: %w", err)
	}

	conn, err := dbPool.Acquire(context.Background())
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	instance := &SQLConnection{
		PgSQL:   dbPool,
		PgConn:  conn,
		Timeout: timeout,
	}
	pgSQL = instance

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err = dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed ping: %w", err)
	}

	return instance, nil
}

// Ping выполняет ping базы данных.
func (*SQLConnection) Ping() error {
	if pgSQL != nil && pgSQL.PgSQL != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := pgSQL.PgSQL.Ping(ctx)
		if err != nil {
			return fmt.Errorf(`failed ping: %w`, err)
		}
	} else {
		return fmt.Errorf(`database connection pool is closed`)
	}
	return nil
}

// CloseSQLInstance закрывает пул соединений и сбрасывает singleton.
func (*SQLConnection) CloseSQLInstance() {
	if pgSQL != nil && pgSQL.PgSQL != nil {
		logger.Log.Info(&message.LogMessage{Message: "Closing database connection pool..."})
		pgSQL.PgSQL.Close()
		pgSQL = nil // Сбрасываем инстанс
		logger.Log.Info(&message.LogMessage{Message: "Database connection pool closed."})
	}
}
