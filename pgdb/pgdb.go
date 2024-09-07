package pgdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Connection struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Connection, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, errors.Errorf("failed to connect to DB: %v", err)
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, errors.Errorf("failed to ping DB: %v", err)
	}
	return &Connection{
		pool: pool,
	}, nil
}

func (p *Connection) Pool() *pgxpool.Pool {
	return p.pool
}

func (p *Connection) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Connection) Close() {
	p.pool.Close()
}
