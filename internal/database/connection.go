package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/navisha/spark/internal/config"
)

// DB wraps sqlx.DB with additional context.
type DB struct {
	*sqlx.DB
	config config.DatabaseConfig
	log    *logrus.Entry
}

// New creates a new database connection pool.
func New(cfg config.DatabaseConfig, log *logrus.Entry) (*DB, error) {
	db, err := sqlx.Connect("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection established")

	return &DB{
		DB:     db,
		config: cfg,
		log:    log,
	}, nil
}

// HealthCheck checks if the database is reachable.
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.PingContext(ctx)
}

// Close closes the database connection pool.
func (db *DB) Close() error {
	if err := db.DB.Close(); err != nil {
		db.log.WithError(err).Error("failed to close database connection")
		return err
	}
	db.log.Info("database connection closed")
	return nil
}
