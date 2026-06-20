# Go Patterns & Best Practices

## Overview

Skill ini berisi pattern-pattern umum dalam Go programming yang sering digunakan dalam Navisha Spark. Pattern ini membantu menulis kode yang idiomatic, maintainable, dan performan tinggi.

---

## 1. Error Handling Patterns

### 1.1 Sentinel Errors

```go
// Define sentinel errors untuk domain-specific errors
var (
    ErrQuestionNotFound    = errors.New("question not found")
    ErrCategoryDisabled    = errors.New("category is disabled")
    ErrMaxRetriesReached   = errors.New("max retries reached")
    ErrInvalidLevel        = errors.New("invalid level")
    ErrDatabaseConnection  = errors.New("database connection failed")
)

// Usage
func (r *QuestionRepository) GetByID(ctx context.Context, id int) (*Question, error) {
    var q Question
    err := r.db.GetContext(ctx, &q, "SELECT * FROM questions WHERE id = $1", id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrQuestionNotFound
        }
        return nil, fmt.Errorf("failed to get question: %w", err)
    }
    return &q, nil
}
```

### 1.2 Custom Error Types

```go
// Custom error type dengan additional context
type QuestionNotFoundError struct {
    ID     int
    Category string
}

func (e *QuestionNotFoundError) Error() string {
    return fmt.Sprintf("question not found: id=%d, category=%s", e.ID, e.Category)
}

// Usage
func (r *QuestionRepository) GetRandom(ctx context.Context, category, level string) (*Question, error) {
    var q Question
    err := r.db.GetContext(ctx, &q, query, category, level)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, &QuestionNotFoundError{
                ID:       0,
                Category: category,
            }
        }
        return nil, err
    }
    return &q, nil
}
```

### 1.3 Error Wrapping

```go
// Wrap errors dengan context menggunakan %w
func (s *Service) SendQuestion(ctx context.Context, q *Question) error {
    err := s.telegram.Send(ctx, q)
    if err != nil {
        return fmt.Errorf("failed to send question %d to telegram: %w", q.ID, err)
    }
    return nil
}

// Unwrap errors di caller
err := service.SendQuestion(ctx, q)
if err != nil {
    var notFoundErr *QuestionNotFoundError
    if errors.As(err, &notFoundErr) {
        logrus.WithField("question_id", notFoundErr.ID).Warn("Question not found")
    }
}
```

---

## 2. Repository Pattern

### 2.1 Interface Definition

```go
// Define interface untuk abstraction
type QuestionRepository interface {
    GetByID(ctx context.Context, id int) (*Question, error)
    GetRandom(ctx context.Context, category, level string) (*Question, error)
    Create(ctx context.Context, q *Question) error
    Update(ctx context.Context, q *Question) error
    IncrementTimesSent(ctx context.Context, id int) error
    GetRecentSent(ctx context.Context, hours int) ([]int, error)
}
```

### 2.2 Implementation

```go
type questionRepository struct {
    db *sqlx.DB
}

func NewQuestionRepository(db *sqlx.DB) QuestionRepository {
    return &questionRepository{db: db}
}

func (r *questionRepository) GetByID(ctx context.Context, id int) (*Question, error) {
    var q Question
    query := `
        SELECT id, category, level, question, answer, follow_ups, tags,
               times_sent, last_sent_at, created_at, updated_at
        FROM questions
        WHERE id = $1
    `
    err := r.db.GetContext(ctx, &q, query, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get question by id: %w", err)
    }
    return &q, nil
}

func (r *questionRepository) GetRandom(ctx context.Context, category, level string) (*Question, error) {
    var q Question
    query := `
        SELECT id, category, level, question, answer, follow_ups, tags,
               times_sent, last_sent_at, created_at, updated_at
        FROM questions
        WHERE category = $1
          AND level = $2
          AND id NOT IN (
              SELECT question_id FROM sent_history WHERE sent_at > NOW() - INTERVAL '24 hours'
          )
        ORDER BY times_sent ASC, RANDOM()
        LIMIT 1
    `
    err := r.db.GetContext(ctx, &q, query, category, level)
    if err != nil {
        return nil, fmt.Errorf("failed to get random question: %w", err)
    }
    return &q, nil
}
```

---

## 3. Service Layer Pattern

### 3.1 Service Definition

```go
type QuestionService interface {
    GetQuestionForDelivery(ctx context.Context) (*Question, error)
    MarkAsSent(ctx context.Context, q *Question) error
    GenerateQuestion(ctx context.Context, category, level string) (*Question, error)
}

type questionService struct {
    repo      QuestionRepository
    rotation  RotationEngine
    llm       LLMClient
    logger    *logrus.Entry
}

func NewQuestionService(
    repo QuestionRepository,
    rotation RotationEngine,
    llm LLMClient,
    logger *logrus.Entry,
) QuestionService {
    return &questionService{
        repo:      repo,
        rotation:  rotation,
        llm:       llm,
        logger:    logger,
    }
}
```

### 3.2 Service Implementation

```go
func (s *questionService) GetQuestionForDelivery(ctx context.Context) (*Question, error) {
    // 1. Select category dan level menggunakan rotation engine
    category, level, err := s.rotation.SelectNext(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to select next question: %w", err)
    }

    s.logger.WithFields(logrus.Fields{
        "category": category,
        "level":    level,
    }).Info("Selected category and level")

    // 2. Cari question dari database
    q, err := s.repo.GetRandom(ctx, category, level)
    if err == nil {
        s.logger.WithField("question_id", q.ID).Info("Found question in database")
        return q, nil
    }

    if !errors.Is(err, ErrQuestionNotFound) {
        return nil, err
    }

    // 3. Jika tidak ada, generate via LLM
    s.logger.Info("No question found, generating via LLM")
    q, err = s.llm.GenerateQuestion(ctx, category, level)
    if err != nil {
        return nil, fmt.Errorf("failed to generate question via LLM: %w", err)
    }

    // 4. Simpan ke database
    if err := s.repo.Create(ctx, q); err != nil {
        return nil, fmt.Errorf("failed to save generated question: %w", err)
    }

    s.logger.WithField("question_id", q.ID).Info("Generated and saved new question")
    return q, nil
}
```

---

## 4. Functional Options Pattern

### 4.1 Definition

```go
// Define option type
type Option func *Config

// Config struct
type Config struct {
    MaxRetries    int
    RetryDelays   []time.Duration
    Timeout       time.Duration
    DatabaseURL   string
    TelegramToken string
}

// Option functions
func WithMaxRetries(n int) Option {
    return func(c *Config) {
        c.MaxRetries = n
    }
}

func WithRetryDelays(delays []time.Duration) Option {
    return func(c *Config) {
        c.RetryDelays = delays
    }
}

func WithTimeout(d time.Duration) Option {
    return func(c *Config) {
        c.Timeout = d
    }
}

// Constructor dengan functional options
func NewClient(opts ...Option) *Client {
    c := &Config{
        MaxRetries:  3,
        RetryDelays: []time.Duration{1 * time.Minute, 5 * time.Minute, 15 * time.Minute},
        Timeout:     10 * time.Second,
    }

    for _, opt := range opts {
        opt(c)
    }

    return &Client{config: c}
}

// Usage
client := NewClient(
    WithMaxRetries(5),
    WithTimeout(30 * time.Second),
)
```

---

## 5. Context Propagation Pattern

### 5.1 Always Accept Context as First Parameter

```go
// Good - context as first parameter
func (s *Service) SendQuestion(ctx context.Context, q *Question) error {
    // Use context for API calls
    err := s.telegram.Send(ctx, q)
    if err != nil {
        return err
    }
    return nil
}

// Bad - no context
func (s *Service) SendQuestion(q *Question) error {
    err := s.telegram.Send(q)
    return err
}
```

### 5.2 Context with Timeout

```go
func (s *Service) SendQuestionWithTimeout(ctx context.Context, q *Question) error {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    err := s.telegram.Send(ctx, q)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return fmt.Errorf("telegram send timeout: %w", err)
        }
        return err
    }
    return nil
}
```

### 5.3 Context with Values

```go
type contextKey string

const (
    QuestionIDKey contextKey = "question_id"
    CategoryKey   contextKey = "category"
)

func WithQuestionID(ctx context.Context, id int) context.Context {
    return context.WithValue(ctx, QuestionIDKey, id)
}

func GetQuestionID(ctx context.Context) int {
    if id, ok := ctx.Value(QuestionIDKey).(int); ok {
        return id
    }
    return 0
}

// Usage
ctx = WithQuestionID(ctx, q.ID)
err := s.telegram.Send(ctx, q)
```

---

## 6. Retry Pattern with Exponential Backoff

### 6.1 Basic Retry

```go
func RetryWithBackoff(maxRetries int, delays []time.Duration, fn func() error) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }

        if i < maxRetries-1 {
            delay := delays[i]
            logrus.WithFields(logrus.Fields{
                "attempt": i + 1,
                "delay":   delay,
                "error":   err,
            }).Warn("Retry attempt failed, waiting before retry")
            time.Sleep(delay)
        }
    }
    return fmt.Errorf("max retries (%d) reached: %w", maxRetries, err)
}

// Usage
err := RetryWithBackoff(3, []time.Duration{1*time.Minute, 5*time.Minute, 15*time.Minute}, func() error {
    return telegramClient.Send(ctx, msg)
})
```

### 6.2 Retry with Error Classification

```go
type RetryPolicy struct {
    MaxRetries int
    Delays     []time.Duration
}

func (p *RetryPolicy) ShouldRetry(err error, attempt int) bool {
    if attempt >= p.MaxRetries {
        return false
    }

    // Retry on network errors
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }

    // Retry on HTTP 5xx
    var httpErr *HTTPError
    if errors.As(err, &httpErr) && httpErr.StatusCode >= 500 {
        return true
    }

    // Retry on HTTP 429 (rate limit)
    if errors.As(err, &httpErr) && httpErr.StatusCode == 429 {
        return true
    }

    // Don't retry on HTTP 4xx (except 429)
    return false
}

func (p *RetryPolicy) GetDelay(attempt int) time.Duration {
    if attempt < len(p.Delays) {
        return p.Delays[attempt]
    }
    return p.Delays[len(p.Delays)-1]
}
```

---

## 7. Worker Pool Pattern

### 7.1 Basic Worker Pool

```go
type Job struct {
    Question *Question
}

type Result struct {
    Question *Question
    Error    error
}

func worker(id int, jobs <-chan Job, results chan<- Result, sendFunc func(*Question) error) {
    for job := range jobs {
        err := sendFunc(job.Question)
        results <- Result{
            Question: job.Question,
            Error:    err,
        }
    }
}

func StartWorkerPool(numWorkers int, jobs []Job, sendFunc func(*Question) error) []Result {
    jobsChan := make(chan Job, len(jobs))
    resultsChan := make(chan Result, len(jobs))

    // Start workers
    for w := 0; w < numWorkers; w++ {
        go worker(w, jobsChan, resultsChan, sendFunc)
    }

    // Send jobs
    for _, job := range jobs {
        jobsChan <- job
    }
    close(jobsChan)

    // Collect results
    var results []Result
    for i := 0; i < len(jobs); i++ {
        results = append(results, <-resultsChan)
    }
    return results
}
```

### 7.2 Worker Pool with Context

```go
func workerWithContext(ctx context.Context, id int, jobs <-chan Job, results chan<- Result) {
    for job := range jobs {
        select {
        case <-ctx.Done():
            return
        default:
            err := sendFunc(job.Question)
            results <- Result{
                Question: job.Question,
                Error:    err,
            }
        }
    }
}
```

---

## 8. Graceful Shutdown Pattern

### 8.1 Signal Handling

```go
func main() {
    // Create context that cancels on SIGINT or SIGTERM
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // Initialize services
    app, err := NewApp(ctx)
    if err != nil {
        logrus.Fatal("Failed to initialize app:", err)
    }

    // Start app
    if err := app.Start(); err != nil {
        logrus.Fatal("Failed to start app:", err)
    }

    // Wait for interrupt signal
    <-ctx.Done()

    logrus.Info("Shutting down gracefully...")

    // Shutdown with timeout
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := app.Shutdown(shutdownCtx); err != nil {
        logrus.WithError(err).Error("Error during shutdown")
    }

    logrus.Info("Shutdown complete")
}
```

### 8.2 Graceful Shutdown Implementation

```go
type App struct {
    scheduler *cron.Cron
    db        *sqlx.DB
    telegram  *TelegramClient
    logger    *logrus.Logger
}

func (a *App) Shutdown(ctx context.Context) error {
    var wg sync.WaitGroup
    var errs []error

    // Stop scheduler
    wg.Add(1)
    go func() {
        defer wg.Done()
        a.scheduler.Stop()
        a.logger.Info("Scheduler stopped")
    }()

    // Close database
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := a.db.Close(); err != nil {
            errs = append(errs, fmt.Errorf("failed to close database: %w", err))
        } else {
            a.logger.Info("Database connection closed")
        }
    }()

    // Wait for all shutdown tasks or timeout
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        if len(errs) > 0 {
            return errors.Join(errs...)
        }
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

---

## 9. Singleton Pattern (with sync.Once)

```go
type Database struct {
    db *sqlx.DB
}

var (
    instance *Database
    once     sync.Once
    initErr  error
)

func GetDatabase(connectionString string) (*Database, error) {
    once.Do(func() {
        db, err := sqlx.Connect("postgres", connectionString)
        if err != nil {
            initErr = fmt.Errorf("failed to connect to database: %w", err)
            return
        }

        // Configure connection pool
        db.SetMaxOpenConns(10)
        db.SetMaxIdleConns(5)
        db.SetConnMaxLifetime(5 * time.Minute)

        instance = &Database{db: db}
    })

    return instance, initErr
}

// Reset for testing
func ResetDatabase() {
    once = sync.Once{}
    instance = nil
    initErr = nil
}
```

---

## 10. Middleware Pattern (HTTP)

### 10.1 Logging Middleware

```go
func LoggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Process request
            next.ServeHTTP(w, r)

            // Log after request
            logger.WithFields(logrus.Fields{
                "method":     r.Method,
                "path":       r.URL.Path,
                "status":     w.(*loggingResponseWriter).status,
                "duration":   time.Since(start),
                "remote_addr": r.RemoteAddr,
            }).Info("HTTP request")
        })
    }
}

// Response writer to capture status code
type loggingResponseWriter struct {
    http.ResponseWriter
    status int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
    lrw.status = code
    lrw.ResponseWriter.WriteHeader(code)
}
```

### 10.2 Recovery Middleware

```go
func RecoveryMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if rec := recover(); rec != nil {
                    logger.WithFields(logrus.Fields{
                        "error": rec,
                        "path":  r.URL.Path,
                    }).Error("Panic recovered")

                    http.Error(w, "Internal server error", http.StatusInternalServerError)
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 11. Builder Pattern

```go
type QuestionBuilder struct {
    question *Question
}

func NewQuestionBuilder() *QuestionBuilder {
    return &QuestionBuilder{
        question: &Question{},
    }
}

func (b *QuestionBuilder) Category(category string) *QuestionBuilder {
    b.question.Category = category
    return b
}

func (b *QuestionBuilder) Level(level string) *QuestionBuilder {
    b.question.Level = level
    return b
}

func (b *QuestionBuilder) Question(text string) *QuestionBuilder {
    b.question.Question = text
    return b
}

func (b *QuestionBuilder) Answer(text string) *QuestionBuilder {
    b.question.Answer = text
    return b
}

func (b *QuestionBuilder) FollowUps(followUps []string) *QuestionBuilder {
    b.question.FollowUps = followUps
    return b
}

func (b *QuestionBuilder) Tags(tags []string) *QuestionBuilder {
    b.question.Tags = tags
    return b
}

func (b *QuestionBuilder) Build() *Question {
    b.question.CreatedAt = time.Now()
    b.question.UpdatedAt = time.Now()
    return b.question
}

// Usage
q := NewQuestionBuilder().
    Category("Golang").
    Level("intermediate").
    Question("Apa itu goroutine?").
    Answer("Goroutine adalah thread ringan...").
    FollowUps([]string{"Bagaimana cara membuat goroutine?"}).
    Tags([]string{"golang", "concurrency"}).
    Build()
```

---

## 12. Circuit Breaker Pattern

```go
type CircuitBreaker struct {
    maxFailures  int
    timeout      time.Duration
    failures     int
    lastFailure  time.Time
    state        State
    mu           sync.Mutex
}

type State int

const (
    StateClosed   State = iota // Normal operation
    StateOpen                  // Failing, reject requests
    StateHalfOpen              // Testing if service recovered
)

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        maxFailures: maxFailures,
        timeout:     timeout,
        state:       StateClosed,
    }
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    state := cb.state
    cb.mu.Unlock()

    if state == StateOpen {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.mu.Lock()
            cb.state = StateHalfOpen
            cb.mu.Unlock()
        } else {
            return fmt.Errorf("circuit breaker is open")
        }
    }

    err := fn()
    if err != nil {
        cb.onFailure()
        return err
    }

    cb.onSuccess()
    return nil
}

func (cb *CircuitBreaker) onFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailure = time.Now()

    if cb.failures >= cb.maxFailures {
        cb.state = StateOpen
    }
}

func (cb *CircuitBreaker) onSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0
    cb.state = StateClosed
}
```

---

## 13. Rate Limiter Pattern

```go
type RateLimiter struct {
    rate       time.Duration
    lastCall   time.Time
    maxCalls   int
    calls      int
    mu         sync.Mutex
}

func NewRateLimiter(rate time.Duration, maxCalls int) *RateLimiter {
    return &RateLimiter{
        rate:     rate,
        maxCalls: maxCalls,
    }
}

func (rl *RateLimiter) Wait() {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    elapsed := now.Sub(rl.lastCall)

    if elapsed < rl.rate && rl.calls >= rl.maxCalls {
        sleepTime := rl.rate - elapsed
        time.Sleep(sleepTime)
        rl.calls = 0
        rl.lastCall = time.Now()
    } else if elapsed >= rl.rate {
        rl.calls = 0
        rl.lastCall = now
    }

    rl.calls++
}

// Usage
limiter := NewRateLimiter(1*time.Second, 1) // 1 call per second

for _, q := range questions {
    limiter.Wait()
    err := telegramClient.Send(ctx, q)
    if err != nil {
        logrus.WithError(err).Error("Failed to send question")
    }
}
```

---

## 14. Lazy Initialization Pattern

```go
type Lazy[T any] struct {
    once sync.Once
    val  T
    err  error
    fn   func() (T, error)
}

func NewLazy(fn func() (T, error)) *Lazy[T] {
    return &Lazy[T]{fn: fn}
}

func (l *Lazy[T]) Get() (T, error) {
    l.once.Do(func() {
        l.val, l.err = l.fn()
    })
    return l.val, l.err
}

// Usage
var db = NewLazy(func() (*sqlx.DB, error) {
    return sqlx.Connect("postgres", connectionString)
})

// Later in code
database, err := db.Get()
if err != nil {
    return err
}
```

---

## 15. Common Idioms

### 15.1 Multiple Return Values

```go
// Go idiom: return value + error
func GetUser(id int) (*User, error) {
    user, err := db.GetUser(id)
    if err != nil {
        return nil, err
    }
    return user, nil
}
```

### 15.2 Defer for Cleanup

```go
func ProcessFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close() // Will be called when function returns

    // Process file
    return nil
}
```

### 15.3 Range over Slices/Maps

```go
// Iterate over slice
for i, item := range items {
    logrus.Infof("Item %d: %v", i, item)
}

// Iterate over map
for key, value := range config {
    logrus.Infof("%s: %v", key, value)
}

// Ignore key or value
for _, value := range items {
    fmt.Println(value)
}
```

### 15.4 Type Assertion and Type Switch

```go
// Type assertion
val, ok := interfaceVar.(string)
if !ok {
    return fmt.Errorf("not a string")
}

// Type switch
switch v := interfaceVar.(type) {
case string:
    fmt.Println("String:", v)
case int:
    fmt.Println("Int:", v)
case nil:
    fmt.Println("Nil")
default:
    fmt.Println("Unknown type")
}
```

### 15.5 Variadic Functions

```go
func Sum(nums ...int) int {
    total := 0
    for _, num := range nums {
        total += num
    }
    return total
}

// Usage
sum := Sum(1, 2, 3, 4, 5)
nums := []int{1, 2, 3, 4, 5}
sum := Sum(nums...)
```

---

## 16. Anti-Patterns to Avoid

### ❌ Don't Use init() for Complex Logic

```go
// Bad
func init() {
    db, _ := sqlx.Connect("postgres", connectionString)
    // What if connection fails?
}

// Good - use explicit initialization
func NewApp() (*App, error) {
    db, err := sqlx.Connect("postgres", connectionString)
    if err != nil {
        return nil, err
    }
    return &App{db: db}, nil
}
```

### ❌ Don't Use naked returns in long functions

```go
// Bad - confusing in long functions
func process() (result int, err error) {
    if something {
        return 0, fmt.Errorf("error")
    }
    result = 42
    return // What is being returned?
}

// Good - explicit returns
func process() (int, error) {
    if something {
        return 0, fmt.Errorf("error")
    }
    return 42, nil
}
```

### ❌ Don't Use time.After in Loops

```go
// Bad - leaks goroutine
for {
    select {
    case <-time.After(1 * time.Minute):
        // Do something
    }
}

// Good - use time.Timer
timer := time.NewTimer(1 * time.Minute)
defer timer.Stop()

for {
    select {
    case <-timer.C:
        // Do something
        timer.Reset(1 * time.Minute)
    }
}
```

---

## 17. Testing Patterns

### 17.1 Table-Driven Tests

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

### 17.2 Mocking with Interfaces

```go
// Define interface
type TelegramClient interface {
    Send(ctx context.Context, msg *Message) error
}

// Mock implementation
type MockTelegramClient struct {
    SendFunc func(ctx context.Context, msg *Message) error
}

func (m *MockTelegramClient) Send(ctx context.Context, msg *Message) error {
    return m.SendFunc(ctx, msg)
}

// Usage in test
func TestSendQuestion(t *testing.T) {
    mockClient := &MockTelegramClient{
        SendFunc: func(ctx context.Context, msg *Message) error {
            return nil
        },
    }

    service := NewQuestionService(mockClient, ...)
    err := service.SendQuestion(context.Background(), testQuestion)
    if err != nil {
        t.Errorf("SendQuestion() error = %v", err)
    }
}
```

---

**Document End**