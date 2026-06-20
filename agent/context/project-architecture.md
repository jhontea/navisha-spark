# Navisha Spark — Project Architecture

## Overview

Navisha Spark adalah sistem pembelajaran backend engineering yang mengirimkan insight teknis ke Telegram setiap 3 jam. Sistem ini dibangun dengan arsitektur **Clean Architecture** yang memisahkan concerns menjadi layer-layer yang jelas.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         cmd/spark/main.go                        │
│                         (Entry Point)                            │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      internal/ (Core Logic)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   config/    │  │  scheduler/  │  │     telegram/        │  │
│  │  (Config     │  │  (Cron       │  │  (Telegram Bot API   │  │
│  │   Loading)   │  │   Trigger)   │  │   Client)            │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   content/   │  │  rotation/   │  │     database/        │  │
│  │  (LLM +      │  │  (Topic      │  │  (PostgreSQL         │  │
│  │   Content    │  │   Selection) │  │   Repository)        │  │
│  │   Bank)      │  │              │  │                      │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
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
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  PostgreSQL  │  │  Telegram    │  │   OpenRouter LLM     │  │
│  │  (Supabase)  │  │  Bot API     │  │   (openrouter/owl-   │  │
│  │              │  │              │  │    alpha)             │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Layer Responsibilities

### 1. Entry Point (`cmd/spark/main.go`)
- Initialize application
- Load configuration
- Setup database connection
- Start scheduler
- Handle graceful shutdown

### 2. Config Layer (`internal/config/`)
- Load environment variables (viper)
- Parse YAML configs (categories, schedule)
- Provide typed configuration structs
- Hot-reload support for YAML files

### 3. Scheduler Layer (`internal/scheduler/`)
- Wrapper around `robfig/cron/v3`
- Timezone handling (Asia/Jakarta)
- Active hours validation
- Trigger content delivery job

### 4. Telegram Layer (`internal/telegram/`)
- Telegram Bot API client
- Message formatting (Markdown)
- Send message with retry logic
- Chat ID whitelist validation

### 5. Content Layer (`internal/content/`)
- Insight bank management
- LLM integration (OpenRouter)
- Prompt template management
- Content validation

### 6. Rotation Layer (`internal/rotation/`)
- Weighted round-robin category selection
- Level distribution (beginner/intermediate/advanced)
- Deduplication logic (24h window)
- Spaced repetition heuristic

### 7. Database Layer (`internal/database/`)
- PostgreSQL connection pool (sqlx)
- Repository pattern for data access
- Transaction management
- Migration runner

### 8. Retry Layer (`internal/retry/`)
- Exponential backoff implementation
- Retry policy configuration
- Error classification (retryable vs non-retryable)

## Data Flow

### Normal Flow (Success)
```
1. Scheduler triggers (every 3 hours)
2. Rotation Engine selects category + level
3. Database queries for eligible insight
4. If found: use curated insight
   If not found: generate via LLM
5. Telegram sends message
6. Log delivery to database
7. Update rotation state
```

### Retry Flow (Failure)
```
1. Scheduler triggers
2. Rotation Engine selects question
3. Telegram send fails
4. Retry with exponential backoff (1m, 5m, 15m)
5. If success: log and continue
6. If all retries fail: log error, skip to next schedule
```

## Key Design Decisions

### 1. Why Clean Architecture?
- **Testability:** Each layer can be tested independently
- **Maintainability:** Clear separation of concerns
- **Flexibility:** Easy to swap implementations (e.g., Telegram → Discord)

### 2. Why sqlx instead of GORM?
- **Control:** Full control over SQL queries
- **Performance:** No ORM overhead
- **Transparency:** Explicit queries, easier to debug
- **Type Safety:** Struct mapping without magic

### 3. Why robfig/cron instead of systemd timer?
- **Portability:** Works on any OS (Windows, Linux, macOS)
- **Flexibility:** Easy to change schedule without OS config
- **State Management:** Rotation state in PostgreSQL persists across restarts
- **Simplicity:** Single binary, no external dependencies

### 4. Why PostgreSQL (Supabase)?
- **Managed:** No need to manage database server
- **Reliable:** Auto-backup, high availability
- **Scalable:** Can handle growth if needed
- **Cost-effective:** Free tier sufficient for single user

### 5. Why OpenRouter?
- **Unified API:** Single API for multiple LLM providers
- **Free Models:** Access to free models (Mistral, Llama, Phi-3)
- **Fallback:** Easy to switch models without code changes
- **Cost Control:** Monitor usage via dashboard

## Package Structure

```
internal/
├── config/
│   ├── config.go           # Main config struct
│   ├── loader.go           # Load from env + YAML
│   └── types.go            # Type definitions
│
├── scheduler/
│   ├── scheduler.go        # Cron wrapper
│   ├── job.go              # Job definition
│   └── timezone.go         # Timezone handling
│
├── telegram/
│   ├── client.go           # Telegram API client
│   ├── formatter.go        # Markdown formatter
│   └── whitelist.go        # Chat ID validation
│
├── content/
│   ├── bank.go             # Question bank
│   ├── generator.go        # LLM generator
│   ├── prompt.go           # Prompt templates
│   └── validator.go        # Content validation
│
├── rotation/
│   ├── engine.go           # Rotation logic
│   ├── selector.go         # Category/level selector
│   ├── dedup.go            # Deduplication
│   └── spaced_repetition.go # Heuristic logic
│
├── database/
│   ├── connection.go       # DB connection pool
│   ├── repository/
│   │   ├── question.go     # Question CRUD
│   │   ├── delivery.go     # Delivery log
│   │   ├── rotation.go     # Rotation state
│   │   └── history.go      # Sent history
│   └── migration/
│       └── runner.go       # Migration runner
│
└── retry/
    ├── retry.go            # Retry logic
    ├── backoff.go          # Exponential backoff
    └── policy.go           # Retry policy
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
- Structured JSON format
- Include context (category, level, question_id)
- Log to stdout (Docker logging driver)
- Separate error log file (optional)

## Concurrency Model

### Single-threaded Scheduler
- Cron runs in main goroutine
- Each job execution is sequential
- No concurrent job execution (prevents race conditions)

### Database Connection Pool
- Max 10 connections (configurable)
- Connection timeout: 5 seconds
- Idle connection timeout: 30 seconds

### Telegram API Calls
- Sequential (one at a time)
- Timeout: 10 seconds per request
- Retry with backoff

## Security Considerations

### Secrets Management
- All secrets in environment variables
- Never commit `.env` to Git
- Use `.env.example` for documentation

### Database Security
- Connection via SSL (Supabase pooler)
- No hardcoded credentials
- Connection string from env var

### Telegram Security
- Chat ID whitelist (only allow specific user)
- Bot token in env var
- No webhook (polling not used, only sendMessage)

### LLM Security
- API key in env var
- Rate limiting (prevent abuse)
- Input validation (sanitize prompts)

## Monitoring & Observability

### Health Check
- Endpoint: `GET /healthz`
- Checks: Database connection, Telegram API reachability
- Response: JSON with status and timestamp

### Metrics (Optional)
- Prometheus format
- Metrics: messages_sent, messages_failed, questions_generated
- Endpoint: `GET /metrics`

### Logging
- Structured JSON (logrus)
- Fields: timestamp, level, message, category, question_id, error
- Output: stdout/stderr

## Deployment Model

### Docker Compose
```yaml
services:
  spark:
    build: .
    restart: unless-stopped
    healthcheck: ...
```

### Environment
- Production: Docker Compose on VPS
- Development: Local Go binary with `.env`
- Testing: Docker Compose with test database

### CI/CD (Future)
- GitHub Actions
- Build Docker image
- Push to registry
- Deploy to VPS via SSH

## Scalability Considerations

### Current (Single User)
- Single Go binary
- Single PostgreSQL database
- ~8 messages/day
- Cost: <$10/month

### Future (Multi-User)
- Multiple chat_id whitelist
- Per-user rotation state
- Horizontal scaling (multiple instances)
- Load balancer

### Future (High Volume)
- Message queue (Kafka) for delivery
- Worker pool for LLM generation
- Redis cache for frequent queries
- CDN for static content

## Technology Stack

| Component | Technology | Reason |
|-----------|-----------|--------|
| Language | Go 1.23 | Performance, simplicity, concurrency |
| Database | PostgreSQL (Supabase) | Managed, reliable, scalable |
| Scheduler | robfig/cron v3 | Flexible, in-process |
| Telegram | go-telegram-bot-api/v6 | Official wrapper |
| LLM | OpenRouter | Unified API, free models |
| Config | Viper + YAML | Hot-reload, flexible |
| Logging | Logrus | Structured logging |
| Database Driver | sqlx + lib/pq | Type-safe, performant |
| Deployment | Docker Compose | Simple, portable |

## Future Enhancements

### v2.0
- Web dashboard for question management
- Multi-user support
- Analytics dashboard
- Export to Anki/JSON

### v3.0
- Full SRS algorithm (SM-2)
- Image/diagram support
- Voice notes
- Integration with Notion/Obsidian

### v4.0
- Multi-platform (Discord, Slack)
- Community questions
- Gamification (streaks, points)
- AI-powered personalization

---

**Document End**