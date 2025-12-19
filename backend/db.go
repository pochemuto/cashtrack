package cashtrack

import (
	"cashtrack/backend/gen/db"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Db struct {
	conn    *pgxpool.Pool
	Queries *db.Queries
}

type DbConfig struct {
	ConnectionString string
}

func NewPgxPool(ctx context.Context, config DbConfig) (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(ctx, config.ConnectionString)
	if err != nil {
		return nil, err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewDB(conn *pgxpool.Pool) (*Db, error) {
	return &Db{
		conn:    conn,
		Queries: db.New(conn),
	}, nil
}
