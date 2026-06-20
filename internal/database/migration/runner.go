package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Migration represents a database migration.
type Migration struct {
	Version     int
	Description string
	Query       string
}

// Runner handles database migrations.
type Runner struct {
	db  *sqlx.DB
	log *logrus.Entry
}

// NewRunner creates a new migration runner.
func NewRunner(db *sqlx.DB, log *logrus.Entry) *Runner {
	return &Runner{
		db:  db,
		log: log,
	}
}

// Run executes all pending migrations.
func (r *Runner) Run(ctx context.Context, migrations []Migration) error {
	// Create migrations tracking table if not exists
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[int]bool)
	for _, v := range applied {
		appliedMap[v] = true
	}

	// Run pending migrations
	for _, m := range migrations {
		if appliedMap[m.Version] {
			r.log.WithFields(logrus.Fields{
				"version":     m.Version,
				"description": m.Description,
			}).Debug("migration already applied, skipping")
			continue
		}

		r.log.WithFields(logrus.Fields{
			"version":     m.Version,
			"description": m.Description,
		}).Info("applying migration")

		if err := r.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", m.Version, m.Description, err)
		}

		r.log.WithFields(logrus.Fields{
			"version":     m.Version,
			"description": m.Description,
		}).Info("migration applied successfully")
	}

	return nil
}

// RunFile executes a raw SQL file content as a migration.
func (r *Runner) RunFile(ctx context.Context, sqlContent string) error {
	r.log.Info("running migration file")

	_, err := r.db.ExecContext(ctx, sqlContent)
	if err != nil {
		return fmt.Errorf("failed to execute migration file: %w", err)
	}

	r.log.Info("migration file executed successfully")
	return nil
}

// ensureMigrationsTable creates the migrations tracking table if it doesn't exist.
func (r *Runner) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			description VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// getAppliedMigrations returns the list of applied migration versions.
func (r *Runner) getAppliedMigrations(ctx context.Context) ([]int, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`

	var versions []int
	err := r.db.SelectContext(ctx, &versions, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	return versions, nil
}

// applyMigration applies a single migration.
func (r *Runner) applyMigration(ctx context.Context, m Migration) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration query
	if _, err := tx.ExecContext(ctx, m.Query); err != nil {
		return fmt.Errorf("failed to execute migration query: %w", err)
	}

	// Record migration
	recordQuery := `INSERT INTO schema_migrations (version, description) VALUES ($1, $2)`
	if _, err := tx.ExecContext(ctx, recordQuery, m.Version, m.Description); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// DefaultMigrations returns the default set of migrations for Navisha Spark.
func DefaultMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Initial schema: insights, delivery_log, rotation_state, sent_history",
			Query:       initialSchema,
		},
	}
}

// initialSchema is the SQL for the initial database schema.
const initialSchema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS insights (
    id SERIAL PRIMARY KEY,
    category VARCHAR(100) NOT NULL,
    level VARCHAR(20) NOT NULL CHECK (level IN ('beginner','intermediate','advanced')),
    title VARCHAR(200) NOT NULL,
    insight TEXT NOT NULL,
    key_points TEXT[] DEFAULT '{}',
    code_example TEXT,
    follow_ups JSONB DEFAULT '[]',
    tags TEXT[] DEFAULT '{}',
    times_sent INT DEFAULT 0,
    last_sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_insights_category_level ON insights(category, level);
CREATE INDEX IF NOT EXISTS idx_insights_last_sent_at ON insights(last_sent_at);
CREATE INDEX IF NOT EXISTS idx_insights_tags ON insights USING GIN(tags);

CREATE TABLE IF NOT EXISTS delivery_log (
    id SERIAL PRIMARY KEY,
    insight_id INT REFERENCES insights(id) ON DELETE CASCADE,
    sent_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success','failed','retry')),
    error_message TEXT,
    telegram_message_id BIGINT
);

CREATE INDEX IF NOT EXISTS idx_delivery_log_sent_at ON delivery_log(sent_at);
CREATE INDEX IF NOT EXISTS idx_delivery_log_insight_id ON delivery_log(insight_id);
CREATE INDEX IF NOT EXISTS idx_delivery_log_status ON delivery_log(status);

CREATE TABLE IF NOT EXISTS rotation_state (
    category VARCHAR(100) PRIMARY KEY,
    last_sent_at TIMESTAMP,
    total_sent INT DEFAULT 0,
    last_level VARCHAR(20),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sent_history (
    insight_id INT REFERENCES insights(id) ON DELETE CASCADE,
    sent_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (insight_id, sent_at)
);

CREATE INDEX IF NOT EXISTS idx_sent_history_sent_at ON sent_history(sent_at);
CREATE INDEX IF NOT EXISTS idx_sent_history_insight_id ON sent_history(insight_id);

-- Triggers
CREATE OR REPLACE FUNCTION update_insights_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_insights_updated_at') THEN
        CREATE TRIGGER trigger_insights_updated_at
            BEFORE UPDATE ON insights
            FOR EACH ROW
            EXECUTE FUNCTION update_insights_updated_at();
    END IF;
END $$;

CREATE OR REPLACE FUNCTION update_rotation_state_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_rotation_state_updated_at') THEN
        CREATE TRIGGER trigger_rotation_state_updated_at
            BEFORE UPDATE ON rotation_state
            FOR EACH ROW
            EXECUTE FUNCTION update_rotation_state_updated_at();
    END IF;
END $$;

CREATE OR REPLACE FUNCTION cleanup_old_sent_history()
RETURNS void AS $$
BEGIN
    DELETE FROM sent_history
    WHERE sent_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;
`

// Ensure time is used
var _ = time.Now
