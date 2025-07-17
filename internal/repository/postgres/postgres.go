package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"delayed-notifier/internal/config"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewDbConnection(config *config.Config) (*DB, error) {
	poolCfg := config.Pool
	cfg, err := pgxpool.ParseConfig(config.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config pgx pool: %w", err)
	}

	cfg.MaxConns = int32(poolCfg.MaxConns)
	cfg.MinConns = int32(poolCfg.MinConns)
	cfg.MaxConnLifetime = poolCfg.MaxLifeTime
	cfg.MaxConnIdleTime = poolCfg.MaxIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
