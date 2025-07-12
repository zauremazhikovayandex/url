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

type SQLConnection struct {
	PgSQL   *pgxpool.Pool
	PgConn  *pgxpool.Conn
	Timeout time.Duration
}

var (
	onceSQLConnectionInstance sync.Once
	pgSQL                     *SQLConnection
)

func SQLInstance() (*SQLConnection, error) {
	var initError error
	cfg := config.AppConfig.PGConfig
	timeout := time.Duration(cfg.DBTimeout) * time.Second

	onceSQLConnectionInstance.Do(func() {
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

		pgSQL = &SQLConnection{
			PgSQL:   dbPool,
			PgConn:  conn,
			Timeout: timeout,
		}
	})

	if initError != nil || pgSQL == nil {
		return nil, initError
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := pgSQL.PgSQL.Ping(ctx); err != nil {
		return nil, fmt.Errorf(`failed ping: %w`, err)
	}

	return pgSQL, nil
}

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

func (*SQLConnection) CloseSQLInstance() {
	if pgSQL != nil && pgSQL.PgSQL != nil {
		logger.Log.Info(&message.LogMessage{Message: "Closing database connection pool..."})
		pgSQL.PgSQL.Close()
		pgSQL = nil // Сбрасываем инстанс
		logger.Log.Info(&message.LogMessage{Message: "Database connection pool closed."})
	}
}

func ConnectToDBOnce() error {
	cfg := config.AppConfig.PGConfig
	pgCfg, _ := pgxpool.ParseConfig(cfg.DBConnection)
	dbPool, err := pgxpool.ConnectConfig(context.Background(), pgCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PG: %w", err)
	} else {
		dbPool.Close()
		return nil
	}
}
