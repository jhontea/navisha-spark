# Backend Best Practices — Navisha Spark

## Database & Repository

### 1. Never Use `SELECT *`
Always specify explicit column names in queries:
```go
// ❌ BAD
query := `SELECT * FROM insights WHERE id = $1`

// ✅ GOOD
query := `
    SELECT id, category, level, title, insight, key_points, 
           code_example, follow_ups, tags, times_sent, 
           last_sent_at, created_at, updated_at
    FROM insights WHERE id = $1`
```

**Reasons:**
- Explicit columns document what the query returns
- Prevents breaking changes when columns are added/removed
- Reduces network transfer for unused columns
- Enables better query planning

### 2. Always Use Context
Every database operation must accept `context.Context`:
```go
// ✅ GOOD
func (r *Repository) GetByID(ctx context.Context, id int) (*Model, error) {
    // use ctx in queries
    err := r.db.GetContext(ctx, &model, query, id)
}
```

### 3. Use Named Parameters for Dynamic Queries
For queries with many parameters, use named params or clear formatting:
```go
// ✅ GOOD
query := `
    INSERT INTO insights (category, level, title, insight, key_points, code_example, follow_ups, tags)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, created_at, updated_at`
```

### 4. Proper Error Handling with Sentinels
```go
// ✅ GOOD — handle not-found separately
if err == sql.ErrNoRows {
    return nil, nil  // Not an error, just empty result
}
if err != nil {
    return nil, fmt.Errorf("failed to get insight %d: %w", id, err)
}
```

## Error Handling

### 1. Always Wrap Errors
```go
// ✅ GOOD
return nil, fmt.Errorf("failed to create insight: %w", err)

// ❌ BAD
return nil, err
```

### 2. Use Error Sentinels for Retry/Non-Retryable
```go
var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate entry")

if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

## Logging

### 1. Structured Logging with Fields
```go
// ✅ GOOD
log.WithFields(logrus.Fields{
    "category":   category,
    "insight_id": insight.ID,
    "source":     "bank",
}).Info("insight selected")

// ❌ BAD
log.Printf("selected insight %d for category %s", insight.ID, category)
```

### 2. Log Levels Discipline
- `Debug` — Development details, high volume
- `Info` — Important events (startup, delivery success, config reload)
- `Warn` — Recoverable issues (retry attempt, validation warning)
- `Error` — Failures needing investigation (delivery failure after retries)

## Configuration

### 1. Layered Configuration
Load config in order: Defaults → Environment → YAML files → CLI flags
```go
setDefaults()       // 1. Hardcoded defaults
loadFromEnv()       // 2. Override from env vars
loadFromYAML()      // 3. Override from YAML files
```

### 2. Validate on Load
Always validate configuration after loading:
```go
func LoadCategoryConfig(path string) (*CategoryConfig, error) {
    // ... load and parse ...
    if len(cfg.Categories) == 0 {
        return nil, fmt.Errorf("no categories defined")
    }
    return &cfg, nil
}
```

## Concurrency

### 1. Use sync.RWMutex for Read-Heavy Operations
```go
type Whitelist struct {
    mu    sync.RWMutex
    chats map[int64]bool
}

func (w *Whitelist) IsAllowed(id int64) bool {
    w.mu.RLock()
    defer w.mu.RUnlock()
    return w.chats[id]
}
```

### 2. Context Cancellation for Long Operations
```go
// ✅ GOOD
func (j *SendInsightJob) ExecuteWithTimeout(timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    return j.Execute(ctx)
}
```

## Testing

### 1. Repository Tests Use Test Containers
```go
func TestInsightRepository(t *testing.T) {
    db := setupTestDB(t)  // Spin up test PostgreSQL
    defer db.Close()
    
    repo := NewInsightRepository(db.DB)
    // ... test ...
}
```

### 2. Mock External Dependencies
```go
type MockTelegramClient struct {
    SendMessageFunc func(ctx context.Context, text string) (int64, error)
}

func (m *MockTelegramClient) SendMessage(ctx context.Context, text string) (int64, error) {
    return m.SendMessageFunc(ctx, text)
}
```

## Security

### 1. Never Log Secrets
```go
// ❌ BAD
log.WithField("token", cfg.Telegram.BotToken).Info("initialized")

// ✅ GOOD
log.WithField("bot_name", api.Self.UserName).Info("telegram bot initialized")
```

### 2. Use Parameterized Queries (Not String Concatenation)
```go
// ✅ GOOD
query := `SELECT * FROM insights WHERE category = $1 AND level = $2`
db.QueryContext(ctx, query, category, level)

// ❌ BAD
db.QueryContext(ctx, "SELECT * FROM insights WHERE category = '" + category + "'")
```

## Performance

### 1. Connection Pool Configuration
```go
db.SetMaxOpenConns(25)    // Max simultaneous connections
db.SetMaxIdleConns(10)    // Keep idle connections ready
db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections
```

### 2. Use LIMIT for Pagination
Always limit result sets:
```go
query := `SELECT ... ORDER BY last_sent_at ASC NULLS FIRST LIMIT 10`
```

### 3. Use Appropriate Indexes
Ensure indexes match query patterns:
```sql
CREATE INDEX idx_insights_category_level ON insights(category, level);
CREATE INDEX idx_insights_last_sent_at ON insights(last_sent_at);