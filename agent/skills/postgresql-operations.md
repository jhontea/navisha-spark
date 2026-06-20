# PostgreSQL Operations — Skill Guide

## Overview

Skill ini berisi panduan lengkap untuk operasi PostgreSQL dalam Navisha Spark. Mencakup connection management, query patterns, transactions, indexing, dan best practices untuk performa.

---

## 1. Connection Management

### 1.1 Database Connection Setup

```go
import (
    "database/sql"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "logrus"
    "time"
)

func NewDatabase(connectionString string) (*sqlx.DB, error) {
    db, err := sqlx.Connect("postgres", connectionString)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(10)                  // Max 10 concurrent connections
    db.SetMaxIdleConns(5)                   // Keep 5 idle connections
    db.SetConnMaxLifetime(5 * time.Minute)  // Recreate connections after 5 min
    db.SetConnMaxIdleTime(30 * time.Second) // Close idle connections after 30s

    // Test connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    logrus.Info("Database connection established")
    return db, nil
}
```

### 1.2 Connection String Format

```
postgresql://username:password@host:port/database?sslmode=require
```

**Navisha Spark Connection String:**
```
url supabase
```

**Parameters:**
- `sslmode=require` — Required untuk Supabase (SSL/TLS)
- `pool_mode=transaction` — Supabase pooler mode (port 6543)
- `connect_timeout=10` — Connection timeout in seconds

### 1.3 Connection Pool Best Practices

```go
// ✅ Good - reasonable pool settings
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)

// ❌ Bad - too many connections
db.SetMaxOpenConns(100)  // Can overwhelm database

// ❌ Bad - no idle connections
db.SetMaxIdleConns(0)    // Every query creates new connection
```

---

## 2. Basic Queries

### 2.1 Select Single Row

```go
func (r *QuestionRepository) GetByID(ctx context.Context, id int) (*Question, error) {
    var q Question
    query := `
        SELECT id, category, level, question, answer, follow_ups, tags,
               times_sent, last_sent_at, created_at, updated_at
        FROM questions
        WHERE id = $1
    `
    err := r.db.GetContext(ctx, &q, query, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrQuestionNotFound
        }
        return nil, fmt.Errorf("failed to get question: %w", err)
    }
    return &q, nil
}
```

### 2.2 Select Multiple Rows

```go
func (r *QuestionRepository) GetByCategory(ctx context.Context, category string) ([]Question, error) {
    var questions []Question
    query := `
        SELECT id, category, level, question, answer, follow_ups, tags,
               times_sent, last_sent_at, created_at, updated_at
        FROM questions
        WHERE category = $1
        ORDER BY created_at DESC
    `
    err := r.db.SelectContext(ctx, &questions, query, category)
    if err != nil {
        return nil, fmt.Errorf("failed to get questions: %w", err)
    }
    return questions, nil
}
```

### 2.3 Insert

```go
func (r *QuestionRepository) Create(ctx context.Context, q *Question) error {
    query := `
        INSERT INTO questions (category, level, question, answer, follow_ups, tags)
        VALUES (:category, :level, :question, :answer, :follow_ups, :tags)
        RETURNING id, created_at, updated_at
    `
    namedQuery, args, err := sqlx.Named(query, q)
    if err != nil {
        return fmt.Errorf("failed to prepare query: %w", err)
    }

    err = r.db.QueryRowxContext(ctx, namedQuery, args...).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
    if err != nil {
        return fmt.Errorf("failed to insert question: %w", err)
    }
    return nil
}
```

### 2.4 Update

```go
func (r *QuestionRepository) Update(ctx context.Context, q *Question) error {
    query := `
        UPDATE questions
        SET category = :category,
            level = :level,
            question = :question,
            answer = :answer,
            follow_ups = :follow_ups,
            tags = :tags,
            updated_at = NOW()
        WHERE id = :id
    `
    _, err := r.db.NamedExecContext(ctx, query, q)
    if err != nil {
        return fmt.Errorf("failed to update question: %w", err)
    }
    return nil
}
```

### 2.5 Delete

```go
func (r *QuestionRepository) Delete(ctx context.Context, id int) error {
    query := `DELETE FROM questions WHERE id = $1`
    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete question: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrQuestionNotFound
    }

    return nil
}
```

---

## 3. Advanced Queries

### 3.1 Random Question with Deduplication

```go
func (r *QuestionRepository) GetRandomWithDedup(
    ctx context.Context,
    category, level string,
    excludeHours int,
) (*Question, error) {
    var q Question
    query := `
        SELECT id, category, level, question, answer, follow_ups, tags,
               times_sent, last_sent_at, created_at, updated_at
        FROM questions
        WHERE category = $1
          AND level = $2
          AND id NOT IN (
              SELECT question_id
              FROM sent_history
              WHERE sent_at > NOW() - INTERVAL '1 hour' * $3
          )
        ORDER BY times_sent ASC, RANDOM()
        LIMIT 1
    `
    err := r.db.GetContext(ctx, &q, query, category, level, excludeHours)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrQuestionNotFound
        }
        return nil, fmt.Errorf("failed to get random question: %w", err)
    }
    return &q, nil
}
```

### 3.2 Weighted Round-Robin Query

```go
func (r *RotationRepository) GetNextCategory(ctx context.Context) (string, error) {
    var category string
    query := `
        SELECT rs.category
        FROM rotation_state rs
        ORDER BY (EXTRACT(EPOCH FROM (NOW() - rs.last_sent_at)) * COALESCE(c.weight, 1.0)) DESC
        LIMIT 1
    `
    err := r.db.GetContext(ctx, &category, query)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            // Fallback: return first enabled category
            return r.getFirstEnabledCategory(ctx)
        }
        return nil, fmt.Errorf("failed to get next category: %w", err)
    }
    return category, nil
}
```

### 3.3 Bulk Insert

```go
func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []Question) error {
    if len(questions) == 0 {
        return nil
    }

    query := `
        INSERT INTO questions (category, level, question, answer, follow_ups, tags)
        VALUES (:category, :level, :question, :answer, :follow_ups, :tags)
    `

    // Use transaction for bulk insert
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    for _, q := range questions {
        _, err := tx.NamedExecContext(ctx, query, q)
        if err != nil {
            return fmt.Errorf("failed to insert question: %w", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

### 3.4 Upsert (Insert or Update)

```go
func (r *RotationRepository) Upsert(ctx context.Context, state *RotationState) error {
    query := `
        INSERT INTO rotation_state (category, last_sent_at, total_sent, last_level)
        VALUES (:category, :last_sent_at, :total_sent, :last_level)
        ON CONFLICT (category) DO UPDATE SET
            last_sent_at = EXCLUDED.last_sent_at,
            total_sent = rotation_state.total_sent + 1,
            last_level = EXCLUDED.last_level,
            updated_at = NOW()
    `
    _, err := r.db.NamedExecContext(ctx, query, state)
    if err != nil {
        return fmt.Errorf("failed to upsert rotation state: %w", err)
    }
    return nil
}
```

---

## 4. Transactions

### 4.1 Basic Transaction

```go
func (s *Service) SendQuestionAndLog(ctx context.Context, q *Question) error {
    // Begin transaction
    tx, err := s.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Auto-rollback if not committed

    // 1. Send message via Telegram
    err = s.telegram.Send(ctx, q)
    if err != nil {
        return fmt.Errorf("failed to send message: %w", err)
    }

    // 2. Log delivery
    _, err = tx.NamedExecContext(ctx, `
        INSERT INTO delivery_log (question_id, status, telegram_message_id)
        VALUES (:question_id, 'success', :telegram_message_id)
    `, deliveryLog)
    if err != nil {
        return fmt.Errorf("failed to log delivery: %w", err)
    }

    // 3. Update rotation state
    _, err = tx.NamedExecContext(ctx, `
        UPDATE rotation_state
        SET last_sent_at = NOW(), total_sent = total_sent + 1, last_level = :last_level
        WHERE category = :category
    `, rotationState)
    if err != nil {
        return fmt.Errorf("failed to update rotation state: %w", err)
    }

    // 4. Add to sent history
    _, err = tx.ExecContext(ctx, `
        INSERT INTO sent_history (question_id) VALUES ($1)
    `, q.ID)
    if err != nil {
        return fmt.Errorf("failed to add to sent history: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

### 4.2 Transaction with Savepoint

```go
func (s *Service) ComplexOperation(ctx context.Context) error {
    tx, err := s.db.BeginTxx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Step 1
    if err := step1(tx); err != nil {
        return err
    }

    // Create savepoint
    savepoint, err := tx.Exec("SAVEPOINT sp1")
    if err != nil {
        return err
    }

    // Step 2 (might fail)
    if err := step2(tx); err != nil {
        // Rollback to savepoint
        _, err = tx.Exec("ROLLBACK TO SAVEPOINT sp1")
        if err != nil {
            return err
        }
        // Continue with alternative
        if err := step2Alternative(tx); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

---

## 5. JSONB Operations

### 5.1 Store JSONB

```go
func (r *QuestionRepository) Create(ctx context.Context, q *Question) error {
    query := `
        INSERT INTO questions (category, level, question, answer, follow_ups, tags)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6::text[])
    `
    _, err := r.db.ExecContext(ctx, query,
        q.Category,
        q.Level,
        q.Question,
        q.Answer,
        pq.Array(q.FollowUps),  // Convert to JSONB
        pq.Array(q.Tags),       // Convert to text[]
    )
    return err
}
```

### 5.2 Query JSONB

```go
// Find questions with specific follow-up
func (r *QuestionRepository) GetByFollowUpKeyword(ctx context.Context, keyword string) ([]Question, error) {
    var questions []Question
    query := `
        SELECT id, category, level, question, answer, follow_ups, tags
        FROM questions
        WHERE follow_ups @> $1::jsonb
    `
    // @> means "contains" operator
    // $1::jsonb = '["goroutine"]'::jsonb
    err := r.db.SelectContext(ctx, &questions, query, fmt.Sprintf(`["%s"]`, keyword))
    return questions, err
}

// Extract JSONB field
func (r *QuestionRepository) GetFollowUps(ctx context.Context, id int) ([]string, error) {
    var followUps []string
    query := `
        SELECT jsonb_array_elements_text(follow_ups)
        FROM questions
        WHERE id = $1
    `
    err := r.db.SelectContext(ctx, &followUps, query, id)
    return followUps, err
}
```

### 5.3 Update JSONB

```go
// Add new follow-up
func (r *QuestionRepository) AddFollowUp(ctx context.Context, id int, followUp string) error {
    query := `
        UPDATE questions
        SET follow_ups = jsonb_set(follow_ups, '{#}', $1::jsonb, true)
        WHERE id = $2
    `
    // jsonb_set(path, value, create_missing)
    // '{#}' means append to array
    _, err := r.db.ExecContext(ctx, query, fmt.Sprintf(`"%s"`, followUp), id)
    return err
}
```

---

## 6. Array Operations

### 6.1 Store Arrays

```go
// Using pq.Array helper
tags := []string{"golang", "concurrency", "goroutine"}
_, err := db.ExecContext(ctx, `
    INSERT INTO questions (tags) VALUES ($1)
`, pq.Array(tags))
```

### 6.2 Query Arrays

```go
// Find questions with specific tag
func (r *QuestionRepository) GetByTag(ctx context.Context, tag string) ([]Question, error) {
    var questions []Question
    query := `
        SELECT id, category, level, question, answer
        FROM questions
        WHERE $1 = ANY(tags)
    `
    err := r.db.SelectContext(ctx, &questions, query, tag)
    return questions, err
}

// Find questions with ALL tags
func (r *QuestionRepository) GetByAllTags(ctx context.Context, tags []string) ([]Question, error) {
    var questions []Question
    query := `
        SELECT id, category, level, question, answer
        FROM questions
        WHERE tags @> $1::text[]
    `
    // @> means "contains" operator
    err := r.db.SelectContext(ctx, &questions, query, pq.Array(tags))
    return questions, err
}
```

---

## 7. Indexing

### 7.1 Create Index

```sql
-- B-tree index (default, untuk equality dan range queries)
CREATE INDEX idx_questions_category ON questions(category);

-- Composite index (untuk multiple columns)
CREATE INDEX idx_questions_category_level ON questions(category, level);

-- Partial index (untuk subset data)
CREATE INDEX idx_questions_unsent ON questions(id)
WHERE last_sent_at IS NULL;

-- GIN index (untuk JSONB dan arrays)
CREATE INDEX idx_questions_tags ON questions USING GIN(tags);
CREATE INDEX idx_questions_follow_ups ON questions USING GIN(follow_ups);

-- Expression index
CREATE INDEX idx_questions_lower_category ON questions(LOWER(category));
```

### 7.2 Check Index Usage

```sql
-- List all indexes
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'questions';

-- Check index usage
SELECT indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'questions';

-- Explain query plan
EXPLAIN ANALYZE
SELECT * FROM questions
WHERE category = 'Golang' AND level = 'intermediate';
```

---

## 8. Performance Optimization

### 8.1 Use Prepared Statements

```go
// ✅ Good - prepared statement (reused)
func (r *QuestionRepository) GetByID(ctx context.Context, id int) (*Question, error) {
    stmt, err := r.db.PreparexContext(ctx, `
        SELECT id, category, level, question, answer
        FROM questions
        WHERE id = $1
    `)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    var q Question
    err = stmt.GetContext(ctx, &q, id)
    return &q, err
}
```

### 8.2 Batch Operations

```go
// ✅ Good - batch insert
func BulkInsert(ctx context.Context, db *sqlx.DB, questions []Question) error {
    tx, err := db.BeginTxx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.PreparexContext(ctx, `
        INSERT INTO questions (category, level, question, answer)
        VALUES ($1, $2, $3, $4)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, q := range questions {
        _, err := stmt.ExecContext(ctx, q.Category, q.Level, q.Question, q.Answer)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### 8.3 Avoid N+1 Queries

```go
// ❌ Bad - N+1 query problem
func GetQuestionsWithDeliveryLogs(db *sqlx.DB) ([]Question, error) {
    var questions []Question
    err := db.Select(&questions, "SELECT * FROM questions")
    if err != nil {
        return nil, err
    }

    // N additional queries
    for i := range questions {
        var logs []DeliveryLog
        db.Select(&logs, "SELECT * FROM delivery_log WHERE question_id = $1", questions[i].ID)
        questions[i].DeliveryLogs = logs
    }
    return questions, nil
}

// ✅ Good - single query with JOIN
func GetQuestionsWithDeliveryLogs(db *sqlx.DB) ([]Question, error) {
    var questions []Question
    err := db.Select(&questions, `
        SELECT q.*, dl.id as log_id, dl.sent_at, dl.status
        FROM questions q
        LEFT JOIN delivery_log dl ON q.id = dl.question_id
    `)
    return questions, err
}
```

---

## 9. Common Patterns

### 9.1 Pagination

```go
func (r *QuestionRepository) GetPaginated(
    ctx context.Context,
    category string,
    page, perPage int,
) ([]Question, int, error) {
    // Get total count
    var total int
    err := r.db.GetContext(ctx, &total, `
        SELECT COUNT(*) FROM questions WHERE category = $1
    `, category)
    if err != nil {
        return nil, 0, err
    }

    // Get paginated data
    offset := (page - 1) * perPage
    var questions []Question
    err = r.db.SelectContext(ctx, &questions, `
        SELECT id, category, level, question, answer
        FROM questions
        WHERE category = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `, category, perPage, offset)
    if err != nil {
        return nil, 0, err
    }

    return questions, total, nil
}
```

### 9.2 Full-Text Search

```sql
-- Enable full-text search
ALTER TABLE questions ADD COLUMN search_vector tsvector;

UPDATE questions
SET search_vector = 
    setweight(to_tsvector('english', coalesce(question, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(answer, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(array_to_string(tags, ' '), '')), 'C');

-- Create index
CREATE INDEX idx_questions_search ON questions USING GIN(search_vector);

-- Trigger to auto-update
CREATE TRIGGER tsvectorupdate BEFORE INSERT OR UPDATE
ON questions FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger(search_vector, 'pg_catalog.english', question, answer, tags);
```

```go
// Search function
func (r *QuestionRepository) Search(ctx context.Context, query string) ([]Question, error) {
    var questions []Question
    err := r.db.SelectContext(ctx, &questions, `
        SELECT id, category, level, question, answer
        FROM questions
        WHERE search_vector @@ plainto_tsquery('english', $1)
        ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
        LIMIT 20
    `, query)
    return questions, err
}
```

### 9.3 Locking

```go
// Row-level lock (FOR UPDATE)
func (r *QuestionRepository) GetForUpdate(ctx context.Context, id int) (*Question, error) {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    var q Question
    err = tx.GetContext(ctx, &q, `
        SELECT * FROM questions
        WHERE id = $1
        FOR UPDATE
    `, id)
    if err != nil {
        return nil, err
    }

    // Modify q
    q.TimesSent++
    _, err = tx.NamedExecContext(ctx, `
        UPDATE questions SET times_sent = :times_sent WHERE id = :id
    `, q)
    if err != nil {
        return nil, err
    }

    return tx.Commit()
}

// Advisory lock (application-level lock)
func (r *RotationRepository) LockCategory(ctx context.Context, category string) (bool, error) {
    var acquired bool
    err := r.db.GetContext(ctx, &acquired, `
        SELECT pg_try_advisory_xact_lock(hashtext($1))
    `, category)
    return acquired, err
}
```

---

## 10. Error Handling

### 10.1 Common PostgreSQL Errors

```go
import (
    "github.com/lib/pq"
)

func HandlePostgresError(err error) error {
    if err == nil {
        return nil
    }

    var pgErr *pq.Error
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // unique_violation
            return fmt.Errorf("duplicate entry: %w", err)
        case "23503": // foreign_key_violation
            return fmt.Errorf("foreign key violation: %w", err)
        case "23502": // not_null_violation
            return fmt.Errorf("required field missing: %s", pgErr.Column)
        case "42P01": // undefined_table
            return fmt.Errorf("table does not exist: %w", err)
        default:
            return fmt.Errorf("postgres error %s: %w", pgErr.Code, err)
        }
    }

    return err
}
```

### 10.2 Retry on Connection Failure

```go
func RetryOnConnectionError(maxRetries int, fn func() error) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }

        // Check if error is retryable
        var pgErr *pq.Error
        if errors.As(err, &pgErr) {
            // Retry on connection errors
            if pgErr.Code == "57P01" || // admin_shutdown
               pgErr.Code == "57P02" || // crash_shutdown
               pgErr.Code == "57P03" || // cannot_connect_now
               pgErr.Code == "53300" { // too_many_connections
                time.Sleep(time.Duration(i+1) * time.Second)
                continue
            }
        }

        // Non-retryable error
        return err
    }
    return fmt.Errorf("max retries (%d) reached: %w", maxRetries, err)
}
```

---

## 11. Migrations

### 11.1 Manual Migration Runner

```go
func RunMigrations(db *sqlx.DB, migrationDir string) error {
    files, err := os.ReadDir(migrationDir)
    if err != nil {
        return err
    }

    for _, file := range files {
        if !strings.HasSuffix(file.Name(), ".sql") {
            continue
        }

        path := filepath.Join(migrationDir, file.Name())
        content, err := os.ReadFile(path)
        if err != nil {
            return fmt.Errorf("failed to read migration %s: %w", file.Name(), err)
        }

        logrus.WithField("migration", file.Name()).Info("Running migration")

        _, err = db.Exec(string(content))
        if err != nil {
            return fmt.Errorf("failed to run migration %s: %w", file.Name(), err)
        }
    }

    logrus.Info("All migrations completed")
    return nil
}
```

### 11.2 Migration Tracking

```sql
-- Create migrations table
CREATE TABLE IF NOT EXISTS migrations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT NOW()
);

-- Run migration
INSERT INTO migrations (name) VALUES ('001_init.sql')
ON CONFLICT (name) DO NOTHING;
```

---

## 12. Monitoring

### 12.1 Query Statistics

```go
func LogQueryStats(db *sqlx.DB) {
    var stats struct {
        TotalQueries    int
        ActiveQueries   int
        TotalRows       int
    }

    db.Get(&stats, `
        SELECT
            sum(calls) as total_queries,
            sum(active_time) as active_queries,
            sum(rows) as total_rows
        FROM pg_stat_statements
        WHERE query LIKE '%questions%'
    `)

    logrus.WithFields(logrus.Fields{
        "total_queries":  stats.TotalQueries,
        "active_queries": stats.ActiveQueries,
        "total_rows":     stats.TotalRows,
    }).Info("Database query statistics")
}
```

### 12.2 Connection Pool Stats

```go
func LogConnectionPoolStats(db *sqlx.DB) {
    stats := db.Stats()

    logrus.WithFields(logrus.Fields{
        "max_open":     stats.MaxOpenConnections,
        "open":         stats.OpenConnections,
        "in_use":       stats.InUse,
        "idle":         stats.Idle,
        "wait_count":   stats.WaitCount,
        "wait_duration": stats.WaitDuration,
    }).Info("Connection pool statistics")
}
```

---

## 13. Best Practices

### 13.1 Do's

✅ **Always use context** untuk cancellation dan timeout  
✅ **Use transactions** untuk multiple related operations  
✅ **Use prepared statements** untuk repeated queries  
✅ **Use connection pooling** untuk performance  
✅ **Index frequently queried columns**  
✅ **Use JSONB** untuk flexible schema  
✅ **Handle errors properly** dengan wrapping  
✅ **Use sqlx Named queries** untuk readability  
✅ **Close rows and statements** dengan defer  
✅ **Monitor query performance** dengan EXPLAIN

### 13.2 Don'ts

❌ **Don't use string concatenation** untuk queries (SQL injection risk)  
❌ **Don't ignore errors**  
❌ **Don't use SELECT *** in production  
❌ **Don't forget to close connections**  
❌ **Don't use transactions** untuk single queries  
❌ **Don't over-index** (slows down writes)  
❌ **Don't store sensitive data** tanpa encryption  
❌ **Don't use raw database/sql** when sqlx is available

---

## 14. SQL Injection Prevention

### 14.1 Parameterized Queries

```go
// ✅ Good - parameterized query
db.Get(&q, "SELECT * FROM questions WHERE id = $1", id)

// ❌ Bad - SQL injection vulnerability
db.Get(&q, fmt.Sprintf("SELECT * FROM questions WHERE id = %d", id))
```

### 14.2 Named Queries

```go
// ✅ Good - named parameters
db.NamedExec(`
    INSERT INTO questions (category, level)
    VALUES (:category, :level)
`, map[string]interface{}{
    "category": category,
    "level":    level,
})
```

---

## 15. Supabase Specific

### 15.1 Connection via Supabase Pooler

```go
// Supabase transaction mode pooler
connectionString := "url supabase"

db, err := sqlx.Connect("postgres", connectionString)
```

### 15.2 Supabase Features

- **Auto-backup:** Daily backups (free tier)
- **SSL/TLS:** Required (sslmode=require)
- **Pooler:** Port 6543 (transaction mode)
- **Direct connection:** Port 5432 (session mode, limited)

### 15.3 Supabase Dashboard

- **SQL Editor:** Run queries directly
- **Table Editor:** View/edit data
- **Logs:** Query logs dan performance
- **Backups:** Download backups

---

**Document End**