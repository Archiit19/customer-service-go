package db

import (
	"context"
	"fmt"
	"time"

	"github.com/Archiit19/customer-service-go/internal/config"
	"github.com/Archiit19/customer-service-go/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg *config.Config, log logger.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Error(ctx, "parse pool config failed", logger.Err(err))
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MinConns = cfg.DBMinConns
	poolCfg.MaxConnIdleTime = cfg.DBMaxIdleTime
	poolCfg.HealthCheckPeriod = 30 * time.Second

	log.Info(ctx, "creating database pool", logger.String("host", cfg.DBHost), logger.String("port", cfg.DBPort), logger.String("database", cfg.DBName), logger.Int32("max_conns", cfg.DBMaxConns), logger.Int32("min_conns", cfg.DBMinConns), logger.Duration("max_idle_time", cfg.DBMaxIdleTime))
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Error(ctx, "database pool creation failed", logger.Err(err))
		return nil, fmt.Errorf("new pool: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		log.Error(ctx, "database ping failed", logger.Err(err))
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Info(ctx, "database pool ready")
	return pool, nil
}
