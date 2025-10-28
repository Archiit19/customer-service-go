package db

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/yourname/customer-service/internal/config"
)

func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
    dsn := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=%s",
        cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
    )

    poolCfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("parse pool config: %w", err)
    }
    poolCfg.MaxConns = cfg.DBMaxConns
    poolCfg.MinConns = cfg.DBMinConns
    poolCfg.MaxConnIdleTime = cfg.DBMaxIdleTime
    poolCfg.HealthCheckPeriod = 30 * time.Second

    pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
    if err != nil {
        return nil, fmt.Errorf("new pool: %w", err)
    }

    ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    if err := pool.Ping(ctxPing); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping db: %w", err)
    }

    return pool, nil
}
