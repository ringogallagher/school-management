package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Database holds the database connection
type Database struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 seconds
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &Database{conn: db}, nil
}

// Conn returns the underlying database connection
func (d *Database) Conn() *sql.DB {
	return d.conn
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.conn.Close()
}

// BeginTx starts a new transaction
func (d *Database) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.conn.BeginTx(ctx, nil)
}
