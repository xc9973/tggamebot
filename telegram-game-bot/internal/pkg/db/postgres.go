// Package db provides PostgreSQL database connection management.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"telegram-game-bot/internal/config"
)

// Pool wraps pgxpool.Pool with additional functionality.
type Pool struct {
	*pgxpool.Pool
}

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, cfg *config.DatabaseConfig) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(cfg.PoolSize)
	poolConfig.MinConns = int32(cfg.PoolSize / 4) // 25% of max as minimum
	if poolConfig.MinConns < 1 {
		poolConfig.MinConns = 1
	}

	// Connection timeouts
	if cfg.ConnectTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	} else {
		poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
	}

	// Connection lifetime settings
	if cfg.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	} else {
		poolConfig.MaxConnLifetime = time.Hour
	}

	if cfg.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	} else {
		poolConfig.MaxConnIdleTime = 30 * time.Minute
	}

	// Health check settings
	poolConfig.HealthCheckPeriod = 30 * time.Second

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Name).
		Int("pool_size", cfg.PoolSize).
		Msg("Connecting to PostgreSQL")

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("Successfully connected to PostgreSQL")

	return &Pool{Pool: pool}, nil
}

// Close closes the connection pool.
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
		log.Info().Msg("PostgreSQL connection pool closed")
	}
}

// Stats returns pool statistics for monitoring.
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Pool.Stat()
}

// HealthCheck performs a health check on the database connection.
func (p *Pool) HealthCheck(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}

// WithTimeout creates a context with the specified timeout.
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
