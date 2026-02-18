// Package database provides MySQL connection setup using sqlx.
package database

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver registration
	"github.com/jmoiron/sqlx"
)

// DB wraps sqlx.DB.
type DB struct {
	*sqlx.DB
}

// Config holds database connection parameters.
type Config struct {
	DSN                string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
}

// Connect opens a MySQL connection pool and verifies connectivity.
func Connect(ctx context.Context, cfg Config) (*DB, error) {
	db, err := sqlx.ConnectContext(ctx, "mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connecting to MySQL: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute)

	// Verify the connection is alive.
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging MySQL: %w", err)
	}

	return &DB{db}, nil
}

// Ping checks the database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.PingContext(ctx)
}
