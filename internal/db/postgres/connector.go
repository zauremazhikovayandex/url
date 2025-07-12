package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"sync"
	"time"
)

type SqlConnection struct {
	PgSql   *pgxpool.Pool
	PgConn  *pgxpool.Conn
	Timeout time.Duration
}

var (
	onceSqlConnectionInstance sync.Once
	pgSql                     *SqlConnection
)

func SqlInstance() (*SqlConnection, error) {
	var initError error
	cfg := config.AppConfig.PGConfig
	timeout := time.Duration(cfg.DBTimeout) * time.Second

	onceSqlConnectionInstance.Do(func() {
		pgCfg, err := pgxpool.ParseConfig(cfg.DBConnection)
		if err != nil {
			initError = fmt.Errorf("failed to parse PG config: %w", err)
			return
		}

		pgCfg.MaxConnLifetime = 10 * time.Minute
		pgCfg.HealthCheckPeriod = time.Minute

		dbPool, err := pgxpool.ConnectConfig(context.Background(), pgCfg)
		if err != nil {
			initError = fmt.Errorf("failed to connect to PG: %w", err)
			return
		}

		conn, err := dbPool.Acquire(context.Background())
		if err != nil {
			initError = fmt.Errorf("failed to acquire connection: %w", err)
			dbPool.Close() // не забываем закрыть, если Acquire не удался
			return
		}

		pgSql = &SqlConnection{
			PgSql:   dbPool,
			PgConn:  conn,
			Timeout: timeout,
		}
	})

	if initError != nil || pgSql == nil {
		return nil, initError
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := pgSql.PgSql.Ping(ctx); err != nil {
		return nil, fmt.Errorf(`failed ping: %w`, err)
	}

	return pgSql, nil
}

func (*SqlConnection) Ping() error {
	if pgSql != nil && pgSql.PgSql != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := pgSql.PgSql.Ping(ctx)
		if err != nil {
			return fmt.Errorf(`failed ping: %w`, err)
		}
	} else {
		return fmt.Errorf(`database connection pool is closed`)
	}
	return nil
}

func (*SqlConnection) CloseSqlInstance() {
	if pgSql != nil && pgSql.PgSql != nil {
		logger.Log.Info(&message.LogMessage{Message: "Closing database connection pool..."})
		pgSql.PgSql.Close()
		pgSql = nil // Сбрасываем инстанс
		logger.Log.Info(&message.LogMessage{Message: "Database connection pool closed."})
	}
}

func ConnectToDBOnce() error {
	cfg := config.AppConfig.PGConfig
	pgCfg, err := pgxpool.ParseConfig(cfg.DBConnection)
	dbPool, err := pgxpool.ConnectConfig(context.Background(), pgCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PG: %w", err)
	} else {
		dbPool.Close()
		return nil
	}
}
