# Navisha Spark — Coding Standards

## Go Coding Standards

### 1. General Principles

- **Simplicity:** Kode harus mudah dibaca dan dipahami
- **Explicit over Implicit:** Hindari magic, selalu eksplisit
- **Error Handling:** Selalu handle error, jangan panic kecuali truly unrecoverable
- **Concurrency:** Gunakan goroutine dan channel dengan bijak
- **Testing:** Setiap public function harus memiliki unit test

### 2. Naming Conventions

#### Package Names
- Lowercase, single word
- No underscores or mixedCaps
- Example: `database`, `telegram`, `rotation`

#### Function Names
- **Exported:** PascalCase (e.g., `GetQuestion`, `SendMessage`)
- **Unexported:** camelCase (e.g., `getQuestion`, `sendMessage`)
- Verb-first naming: `Get`, `Set`, `Create`, `Update`, `Delete`, `Send`, `Receive`

#### Variable Names
- **Exported:** PascalCase (e.g., `QuestionID`, `MaxRetries`)
- **Unexported:** camelCase (e.g., `questionID`, `maxRetries`)
- Short but descriptive: `i` for index, `err` for error, `ctx` for context

#### Constant Names
- UPPER_SNAKE_CASE
- Example: `const MaxRetries = 3`, `const DefaultTimeout = 5 * time.Second`

#### Interface Names
- Single method interfaces: `-er` suffix (e.g., `Reader`, `Writer`, `Sender`)
- Multi-method interfaces: Descriptive name (e.g., `QuestionRepository`, `TelegramClient`)

### 3. Error Handling

#### Always Check Errors
```go
// Good
result, err := db.GetQuestion(id)
if err != nil {
    return nil, fmt.Errorf("failed to get question: %w", err)
}

// Bad - ignoring error
result, _ := db.GetQuestion(id)
```

#### Error Wrapping
```go
// Use %w to wrap errors for unwrapping
if err != nil {
    return fmt.Errorf("failed to send telegram message: %w", err)
}

// Custom error types for domain errors
type QuestionNotFoundError struct {
    ID int
}

func (e *QuestionNotFoundError) Error() string {
    return fmt.Sprintf("question not found: %d", e.ID)
}
```

#### Sentinel Errors
```go
var (
    ErrQuestionNotFound = errors.New("question not found")
    ErrCategoryDisabled = errors.New("category is disabled")
    ErrMaxRetriesReached = errors.New("max retries reached")
)
```

### 4. Struct Design

#### Use Struct Tags for JSON/DB
```go
type Question struct {
    ID          int       `db:"id" json:"id"`
    Category    string    `db:"category" json:"category"`
    Level       string    `db:"level" json:"level"`
    Question    string    `db:"question" json:"question"`
    Answer      string    `db:"answer" json:"answer"`
    FollowUps   []string  `db:"follow_ups" json:"follow_ups"`
    Tags        []string  `db:"tags" json:"tags"`
    TimesSent   int       `db:"times_sent" json:"times_sent"`
    LastSentAt  *time.Time `db:"last_sent_at" json:"last_sent_at,omitempty"`
    CreatedAt   time.Time `db:"created_at" json:"created_at"`
    UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
```

#### Constructor Functions
```go
func NewQuestion(category, level, question, answer string) *Question {
    return &Question{
        Category: category,
        Level:    level,
        Question: question,
        Answer:   answer,
        FollowUps: []string{},
        Tags:     []string{},
        TimesSent: 0,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

### 5. Function Design

#### Keep Functions Small
- Single responsibility
- Max 20-30 lines per function
- If longer, break into smaller functions

#### Use Functional Options Pattern
```go
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) {
        c.Timeout = d
    }
}

func NewClient(opts ...Option) *Client {
    c := &Config{
        Timeout: 5 * time.Second,
    }
    for _, opt := range opts {
        opt(c)
    }
    return &Client{config: c}
}
```

#### Context Propagation
```go
// Always accept context as first parameter
func (r *QuestionRepository) GetByID(ctx context.Context, id int) (*Question, error) {
    query := `SELECT * FROM questions WHERE id = $1`
    var q Question
    err := r.db.GetContext(ctx, &q, query, id)
    return &q, err
}
```

### 6. Database Access

#### Use sqlx for Queries
```go
// Good - using sqlx
var questions []Question
err := db.Select(&questions, "SELECT * FROM questions WHERE category = $1", category)

// Bad - manual scanning
rows, err := db.Query("SELECT * FROM questions WHERE category = $1", category)
for rows.Next() {
    var q Question
    err := rows.Scan(&q.ID, &q.Category, ...)
    // ...
}
```

#### Use Named Queries for Complex Queries
```go
const getQuestionQuery = `
    SELECT id, category, level, question, answer, follow_ups, tags, times_sent, last_sent_at
    FROM questions
    WHERE category = :category
      AND level = :level
      AND id NOT IN (
          SELECT question_id FROM sent_history WHERE sent_at > NOW() - INTERVAL '24 hours'
      )
    ORDER BY times_sent ASC, RANDOM()
    LIMIT 1
`

func (r *QuestionRepository) GetRandom(ctx context.Context, category, level string) (*Question, error) {
    var q Question
    namedQuery, args, err := sqlx.Named(getQuestionQuery, map[string]interface{}{
        "category": category,
        "level":    level,
    })
    if err != nil {
        return nil, err
    }
    err = r.db.GetContext(ctx, &q, namedQuery, args...)
    return &q, err
}
```

### 7. Concurrency

#### Use Channels for Communication
```go
func worker(id int, jobs <-chan Job, results chan<- Result) {
    for job := range jobs {
        results <- process(job)
    }
}

func startWorkerPool(numWorkers int, jobs []Job) []Result {
    jobsChan := make(chan Job, len(jobs))
    resultsChan := make(chan Result, len(jobs))
    
    for w := 0; w < numWorkers; w++ {
        go worker(w, jobsChan, resultsChan)
    }
    
    for _, job := range jobs {
        jobsChan <- job
    }
    close(jobsChan)
    
    var results []Result
    for i := 0; i < len(jobs); i++ {
        results = append(results, <-resultsChan)
    }
    return results
}
```

#### Use sync.Once for One-time Initialization
```go
var (
    once     sync.Once
    instance *Database
    err      error
)

func GetDatabase() (*Database, error) {
    once.Do(func() {
        instance, err = NewDatabase(connectionString)
    })
    return instance, err
}
```

#### Avoid Shared State
```go
// Bad - shared mutable state
var counter int

func increment() {
    counter++ // Race condition!
}

// Good - use channels or sync primitives
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

### 8. Configuration

#### Use Viper for Config
```go
type Config struct {
    App      AppConfig
    Telegram TelegramConfig
    Database DatabaseConfig
    LLM      LLMConfig
    Schedule ScheduleConfig
}

type AppConfig struct {
    Name  string
    Env   string
    Port  int
    LogLevel string
}

func LoadConfig() (*Config, error) {
    viper.SetConfigFile("config/schedule.yaml")
    viper.SetConfigFile("config/categories.yaml")
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### 9. Logging

#### Use Structured Logging (logrus)
```go
import "github.com/sirupsen/logrus"

func SendQuestion(q *Question) error {
    logrus.WithFields(logrus.Fields{
        "category": q.Category,
        "level":    q.Level,
        "question_id": q.ID,
    }).Info("Sending question to telegram")
    
    err := telegramClient.Send(q)
    if err != nil {
        logrus.WithError(err).WithFields(logrus.Fields{
            "question_id": q.ID,
        }).Error("Failed to send question")
        return err
    }
    
    logrus.Info("Question sent successfully")
    return nil
}
```

#### Log Levels
- **Debug:** Detailed diagnostic information (development only)
- **Info:** General informational messages (startup, shutdown, success)
- **Warn:** Warning messages (retry attempt, fallback used)
- **Error:** Error messages (failed to send, database error)
- **Fatal:** Critical errors (app will exit)

### 10. Testing

#### Table-Driven Tests
```go
func TestGetQuestionByID(t *testing.T) {
    tests := []struct {
        name    string
        id      int
        want    *Question
        wantErr bool
    }{
        {
            name: "valid question",
            id:   1,
            want: &Question{ID: 1, Category: "Golang"},
        },
        {
            name:    "not found",
            id:      999,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := repo.GetByID(context.Background(), tt.id)
            if (err != nil) != tt.wantErr {
                t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && got.ID != tt.want.ID {
                t.Errorf("GetByID() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### Use Test Fixtures
```go
func setupTestDB(t *testing.T) *sqlx.DB {
    db, err := sqlx.Open("postgres", "postgres://localhost:5432/test?sslmode=disable")
    if err != nil {
        t.Fatalf("failed to connect to test db: %v", err)
    }
    
    // Run migrations
    if _, err := db.Exec(migrationSQL); err != nil {
        t.Fatalf("failed to run migrations: %v", err)
    }
    
    // Seed test data
    if _, err := db.Exec(seedSQL); err != nil {
        t.Fatalf("failed to seed test data: %v", err)
    }
    
    return db
}
```

### 11. API Design

#### RESTful Endpoints
```go
// GET /api/v1/questions/:id
func GetQuestion(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    q, err := questionService.GetByID(r.Context(), id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            http.Error(w, "Question not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(q)
}
```

#### Consistent Response Format
```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
    Page       int `json:"page,omitempty"`
    PerPage    int `json:"per_page,omitempty"`
    TotalItems int `json:"total_items,omitempty"`
    TotalPages int `json:"total_pages,omitempty"`
}
```

### 12. Code Organization

#### File Structure
```
package/
├── doc.go              # Package documentation
├── types.go            # Type definitions
├── interface.go        # Interface definitions
├── implementation.go   # Implementation
├── repository.go       # Database access
├── service.go          # Business logic
├── handler.go          # HTTP handlers
└── *_test.go           # Tests
```

#### Package Documentation
```go
// Package rotation provides topic selection and rotation logic
// for Navisha Spark learning scheduler.
//
// It implements weighted round-robin category selection
// and spaced repetition heuristics.
package rotation
```

### 13. Comments

#### Document Public APIs
```go
// GetRandomQuestion retrieves a random question from the database
// based on the specified category and level.
//
// It applies deduplication logic to avoid sending the same question
// within the configured window (default: 24 hours).
//
// Returns ErrQuestionNotFound if no eligible question is available.
func (s *Service) GetRandomQuestion(ctx context.Context, category, level string) (*Question, error) {
    // implementation
}
```

#### Inline Comments (Only When Necessary)
```go
// Calculate priority based on time since last sent
// Longer gap = higher priority
priority := time.Since(r.lastSentAt).Hours() * weight

// Use exponential backoff: 1m, 5m, 15m
delay := time.Duration(math.Pow(2, float64(attempt))) * time.Minute
```

### 14. Anti-Patterns to Avoid

#### ❌ Don't Use Global Variables
```go
// Bad
var db *sqlx.DB

// Good - use dependency injection
type Service struct {
    db *sqlx.DB
}
```

#### ❌ Don't Ignore Errors
```go
// Bad
_ = json.NewEncoder(w).Encode(data)

// Good
if err := json.NewEncoder(w).Encode(data); err != nil {
    http.Error(w, "Failed to encode response", http.StatusInternalServerError)
}
```

#### ❌ Don't Use panic for Control Flow
```go
// Bad
func GetUser(id int) *User {
    user := db.GetUser(id)
    if user == nil {
        panic("user not found")
    }
    return user
}

// Good
func GetUser(id int) (*User, error) {
    user, err := db.GetUser(id)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    return user, nil
}
```

#### ❌ Don't Create God Objects
```go
// Bad
type Manager struct {
    // 50 methods doing everything
}

// Good - separate concerns
type QuestionService struct { ... }
type TelegramService struct { ... }
type RotationService struct { ... }
```

### 15. Code Review Checklist

- [ ] All errors are handled (no `_ = err`)
- [ ] Functions are small and focused (<30 lines)
- [ ] No global variables (use dependency injection)
- [ ] Context is passed to all I/O operations
- [ ] Tests cover happy path and error cases
- [ ] No hardcoded values (use constants or config)
- [ ] Struct tags are correct (db, json)
- [ ] Logging includes relevant context (fields)
- [ ] No panic() except in main() or init()
- [ ] Interfaces are small (1-3 methods)
- [ ] No duplicated code (DRY principle)
- [ ] Naming is clear and consistent

---

**Document End**