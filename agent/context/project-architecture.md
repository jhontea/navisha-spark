# Navisha Spark — Project Architecture

## Overview

Navisha Spark adalah sistem pembelajaran backend engineering yang mengirimkan insight teknis ke Telegram setiap 3 jam. Sistem ini dibangun dengan arsitektur **Clean Architecture** yang memisahkan concerns menjadi layer-layer yang jelas.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         cmd/spark/main.go                        │
│                         (Entry Point)                            │
└────────────────────────────┤────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      internal/ (Core Logic)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │   config/    │  │  scheduler/  │  │     telegram/        │  │
│  │  (Config     │  │  (Cron       │  │  (Telegram Bot API   │  │
│  │   Loading)   │  │   Trigger)   │  │   Client)            │  │
│  └──────────────┘  └──────────────┘  └────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │   content/   │  │  rotation/   │  │     database/        │  │
│  │  (LLM +      │  │  (Topic      │  │  (PostgreSQL         │  │
│  │   Content    │  │   Selection) │  │   Repository)        │  │
│  │   Bank)      │  │              │  │                      │  │
│  └──────────────┘  └──────────────┘  └────────────────────┘  │
│  ┌──────────────┐                                              │
│  │   retry/     │                                              │
│  │  (Exponential│                                              │
│  │   Backoff)   │                                              │
│  └──────────────┘                                              │
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    External Dependencies                         │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │  PostgreSQL  │  │  Telegram    │  │   OpenRouter LLM     │  │
│  │  (Supabase)  │  │  Bot API     │  │   (openrouter/owl-   │  │
│  │              │  │              │  │    alpha)             │  │
│  └──────────────┘  └──────────────┘  └────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Layer Responsibilities

### 1. Entry Point (`cmd/spark/main.go`)
- Initialize application
- Load configuration (`config/config.yaml` + env vars)
- Setup database connection
- Start scheduler
- Start health check HTTP server (`/healthz`, `/trigger`)
- Handle hot-reload via `config.WatchConfig`
- Handle graceful shutdown

### 2. Config Layer (`internal/config/`)
- Load environment variables via `os.Getenv`
- Parse unified YAML config (`config/config.yaml`)
- Provide typed configuration structs
- Hot-reload support via `fsnotify` debounced watcher

### 3. Scheduler Layer (`internal/scheduler/`)
- Wrapper around `robfig/cron/v3`
- Timezone handling (Asia/Jakarta)
- Active hours validation
- `SendInsightJob` — orchestrates the full delivery flow

### 4. Telegram Layer (`internal/telegram/`)
- Telegram Bot API client
- Message formatting (Markdown) with split support (>4096 chars)
- Chat ID whitelist validation

### 5. Content Layer (`internal/content/`)
- LLM integration (OpenRouter REST API, no SDK)
- Prompt builder: insight, variation, follow-up prompts
- Content validator

### 6. Rotation Layer (`internal/rotation/`)
- Weighted round-robin category selection
- Level distribution (beginner/intermediate/advanced)
- Deduplication logic (configurable window)
- Spaced repetition heuristic (SM-2 inspired)

### 7. Database Layer (`internal/database/`)
- PostgreSQL connection pool (sqlx)
- Repository pattern: `InsightRepository`, `RotationRepository`, `HistoryRepository`, `DeliveryRepository`
- Migration runner (`internal/database/migration/runner.go`)

### 8. Retry Layer (`internal/retry/`)
- Exponential backoff, fixed, incremental, and list-based backoff strategies
- Retry policy configuration
- Generic `DoWithData[T]` for type-safe retries

## Data Flow

### Normal Flow (Success)
```
1. Scheduler triggers (every 3 hours)
2. Rotation Engine selects category + level
3. InsightRepository queries for eligible insight (not sent in dedup window)
4. If found: use curated insight from bank
   If not found: generate via LLM (OpenRouter)
5. Validate generated content
6. Save generated insight to database
7. Format message with Telegram Formatter (split if >4096 chars)
8. Send via Telegram Bot API
9. Record delivery: update rotation_state, insights.times_sent, sent_history
```

### Retry Flow (Failure)
```
1. Scheduler triggers
2. Rotation Engine selects insight
3. Telegram send fails
4. Retry with exponential backoff (1m, 5m, 15m)
5. If success: log and record delivery
6. If all retries fail: log error, skip to next schedule
```

## Key Design Decisions

### 1. Why Clean Architecture?
- **Testability:** Each layer can be tested independently
- **Maintainability:** Clear separation of concerns
- **Flexibility:** Easy to swap implementations

### 2. Why sqlx instead of GORM?
- **Control:** Full control over SQL queries
- **Performance:** No ORM overhead
- **Transparency:** Explicit queries, easier to debug

### 3. Why robfig/cron instead of systemd timer?
- **Portability:** Works on any OS
- **State Management:** Rotation state in PostgreSQL persists across restarts
- **Simplicity:** Single binary, no external dependencies

### 4. Why no Viper?
- Viper is not used. Config is loaded via `yaml.v3` + `os.Getenv` directly.
- Lighter approach, fully typed structs, hot-reload via `fsnotify`.

### 5. Why parameterized SQL intervals?
- `($1 * INTERVAL '1 hour')` instead of `($1 || ' hours')::INTERVAL`
- Parameterized prevents SQL injection and is more idiomatic PostgreSQL.

## Package Structure

```
internal/
├── config/
│   ├── config.go           # LoadConfig, WatchConfig, hot-reload
│   ├── loader.go           # GetEnv, GetEnvInt, GetEnvBool, MustConfig
│   └── types.go            # All config struct types
│
├── scheduler/
│   ├── scheduler.go        # Cron wrapper
│   ├── job.go              # SendInsightJob (main orchestration logic)
│   └── timezone.go         # Timezone helpers
│
├── telegram/
│   ├── client.go           # Telegram API client
│   ├── formatter.go        # Markdown formatter + message splitting
│   ├── formatter_test.go   # Unit tests for formatter
│   └── whitelist.go        # Chat ID whitelist
│
├── content/
│   ├── generator.go        # OpenRouter LLM client
│   ├── prompt.go           # Prompt templates (insight, variation, follow-up)
│   └── validator.go        # Content validation & repair
│
├── rotation/
│   ├── engine.go           # Rotation logic (SelectNext, RecordDelivery)
│   ├── selector.go         # Level selection (weighted random)
│   ├── dedup.go            # Deduplication helpers
│   └── spaced_repetition.go # SM-2 inspired heuristics
│
├── database/
│   ├── connection.go       # DB connection pool
│   ├── repository/
│   │   ├── insight.go      # Insight CRUD
│   │   ├── delivery.go     # Delivery log
│   │   ├── rotation.go     # Rotation state
│   │   └── history.go      # Sent history (dedup)
│   └── migration/
│       └── runner.go       # Migration runner
│
└── retry/
    ├── retry.go            # Do, DoWithData[T] generic retry
    ├── backoff.go          # ExponentialBackoff, ListBackoff, FixedBackoff
    └── policy.go           # Policy struct, DefaultRetryableFn
```

## Database Schema

```
insights
├── id, category, level, title, insight
├── key              -- subtopic identifier (kebab-case slug)
├── key_points       -- TEXT[]
├── code_example     -- nullable TEXT
├── follow_ups       -- JSONB [{"q":"...","a":"..."}]
├── tags             -- TEXT[]
├── times_sent, last_sent_at, created_at, updated_at

delivery_log
├── id, insight_id, sent_at, status (success|failed|retry)
└── error_message, telegram_message_id

rotation_state
└── category (PK), last_sent_at, total_sent, last_level, updated_at

sent_history
└── insight_id + sent_at (composite PK) -- deduplication window
```

## Error Handling Strategy

### Retryable Errors
- Network timeout
- HTTP 5xx (server error)
- HTTP 429 (rate limit)
- Database connection temporary failure

### Non-Retryable Errors
- HTTP 4xx (client error, except 429)
- Invalid message format
- Database constraint violation
- Invalid configuration

### Error Logging
- Structured JSON format (logrus)
- Include context: category, level, insight_id, error
- Output: stdout/stderr (Docker logging driver)

## Security Considerations

### Secrets Management
- All secrets in environment variables
- Never commit `.env` to Git
- Use `.env.example` for documentation

### Database Security
- Connection via SSL (Supabase pooler)
- Parameterized queries only (no string interpolation)
- Connection string from env var

### Telegram Security
- Chat ID whitelist (`Whitelist` struct in `telegram/whitelist.go`)
- Bot token in env var
- No webhook (only outbound `sendMessage`)

### HTTP Endpoints
- `/healthz` — public, safe (read-only DB ping)
- `/trigger` — rate-limited via Nginx in production; fires job asynchronously

## Monitoring & Observability

### Health Check
- Endpoint: `GET /healthz`
- Checks: Database connection, Telegram API reachability
- Response: JSON `{"status", "timestamp", "database", "telegram"}`

### Manual Trigger
- Endpoint: `GET /trigger` or `POST /trigger`
- Fires an immediate delivery job (async, non-blocking)
- Returns 202 Accepted immediately

### Logging
- Structured JSON (logrus)
- Fields: timestamp, level, message, category, insight_id, error
- Output: stdout/stderr

---

**Document End**
