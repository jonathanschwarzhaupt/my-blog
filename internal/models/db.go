package models

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnIdleTime time.Duration
}

func OpenPool(ctx context.Context, dsn string, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.MinConns > cfg.MaxConns {
		return nil, fmt.Errorf("models: MinConns (%d) must not exceed MaxConns (%d)", cfg.MinConns, cfg.MaxConns)
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
