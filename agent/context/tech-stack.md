# Navisha Spark — Technology Stack

## Overview

Dokumentasi ini menjelaskan setiap teknologi yang digunakan dalam Navisha Spark, mengapa dipilih, dan bagaimana menggunakannya.

---

## Core Technologies

### 1. Go 1.23

**Purpose:** Primary programming language

**Why Go?**
- **Performance:** Compiled language, performa setara C/C++
- **Simplicity:** Syntax sederhana, mudah dibaca dan dipelajari
- **Concurrency:** Built-in goroutine dan channel
- **Standard Library:** Library yang lengkap (HTTP, JSON, SQL, dll.)
- **Single Binary:** Compile ke single binary, mudah di-deploy
- **Cross-platform:** Compile untuk berbagai OS dan architecture

**Key Features Used:**
- Goroutine untuk concurrent operations
- `context` package untuk cancellation dan timeout
- `net/http` untuk HTTP server (health check + trigger)
- `database/sql` + `sqlx` untuk database access
- Generics (`DoWithData[T]` di retry package)

---

### 2. PostgreSQL (Supabase)

**Purpose:** Primary database — menyimpan insights, delivery logs, dan rotation state

**Why PostgreSQL?**
- **ACID Compliant:** Jamin konsistensi data
- **JSONB Support:** Simpan follow_ups sebagai JSON
- **Array Support:** Simpan tags, key_points sebagai array
- **Managed (Supabase):** Tidak perlu manage database server
- **Free Tier:** Cukup untuk single user (500MB database)

**Key Features Used:**
- `SERIAL` untuk auto-increment primary key
- `JSONB` untuk flexible schema (follow_ups)
- `TEXT[]` untuk array of strings (tags, key_points)
- `CHECK` constraints untuk validasi (level enum)
- `TIMESTAMP DEFAULT NOW()` untuk audit fields
- `ON DELETE CASCADE` untuk referential integrity
- `INTERVAL` arithmetic untuk deduplication window queries

**Connection:**
```
postgresql://username:password@host:6543/postgres (Supabase pooler)
```

---

### 3. robfig/cron v3

**Purpose:** In-process scheduler untuk trigger content delivery setiap 3 jam

**Why robfig/cron?**
- **Standard:** Paling populer untuk cron di Go
- **Flexible:** Support standard cron expressions
- **Lightweight:** Tidak ada dependency berat
- **In-process:** Tidak perlu external process
- **Timezone Support:** Set timezone (Asia/Jakarta)
- **Middleware:** `SkipIfStillRunning`, `Recover` built-in

**Usage:**
```go
import "github.com/robfig/cron/v3"

c := cron.New(
    cron.WithLocation(loc),
    cron.WithChain(
        cron.SkipIfStillRunning(logger),
        cron.Recover(logger),
    ),
)
c.AddFunc("0 */3 * * *", myJob)
c.Start()
```

---

### 4. go-telegram-bot-api (v1 / non-v5)

**Purpose:** Telegram Bot API wrapper untuk Go

**Package:** `github.com/go-telegram-bot-api/telegram-bot-api` (v4.6.4)

**Key Features Used:**
- `tgbotapi.NewBotAPI(token)` untuk create bot client
- `botAPI.Send(msg)` untuk send message
- `tgbotapi.NewMessage(chatID, text)` untuk create message
- `msg.ParseMode = "Markdown"` untuk Markdown formatting

**Usage:**
```go
import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

bot, err := tgbotapi.NewBotAPI(token)
msg := tgbotapi.NewMessage(chatID, text)
msg.ParseMode = "Markdown"
msg.DisableWebPagePreview = true
bot.Send(msg)
```

---

### 5. OpenRouter (via direct HTTP)

**Purpose:** LLM provider untuk generate insights on-the-fly

**Why OpenRouter?**
- **Unified API:** Single API untuk multiple LLM providers
- **Free Models:** `openrouter/owl-alpha` gratis
- **Easy to Switch:** Ganti model via config tanpa ubah code
- **Cost-effective:** Free tier untuk usage kecil

**Implementation:** Direct HTTP call (no SDK) menggunakan `net/http`:
```go
req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
    "https://openrouter.ai/api/v1/chat/completions",
    bytes.NewReader(body))
req.Header.Set("Authorization", "Bearer "+apiKey)
req.Header.Set("Content-Type", "application/json")
```

**Alternatives Considered:**
- OpenAI SDK: Berbayar, overkill
- Anthropic Claude: Berbayar
- Local LLM (Ollama): Butuh GPU

---

### 6. sqlx

**Purpose:** Extension untuk `database/sql` dengan fitur-fitur tambahan

**Why sqlx?**
- **Type-safe:** Auto-map rows ke struct
- **`db:` tags:** Mapping kolom ke field struct
- **`Get` / `Select`:** Query tunggal vs multiple rows
- **Lightweight:** Tidak seperti ORM, tetap kontrol penuh atas SQL

**Usage:**
```go
import "github.com/jmoiron/sqlx"

var insight Insight
err := db.GetContext(ctx, &insight, "SELECT * FROM insights WHERE id = $1", id)

var insights []Insight
err = db.SelectContext(ctx, &insights, "SELECT * FROM insights WHERE category = $1", cat)
```

---

### 7. gopkg.in/yaml.v3

**Purpose:** Parse unified YAML config (`config/config.yaml`)

**Why yaml.v3 directly (no Viper)?**
- **Simpler:** Tidak ada dependency besar
- **Typed:** Langsung unmarshal ke typed struct
- **Hot-reload:** Handled secara manual via `fsnotify`

**Usage:**
```go
import "gopkg.in/yaml.v3"

data, _ := os.ReadFile("config/config.yaml")
yaml.Unmarshal(data, &cfg)
```

---

### 8. fsnotify

**Purpose:** File system watcher untuk hot-reload `config/config.yaml`

**Why fsnotify?**
- **Cross-platform:** Windows, macOS, Linux
- **Low-level:** Minimal overhead
- **Debounced:** Impelmentasi debounce 100ms untuk avoid double-reload

**Usage:**
```go
watcher, _ := fsnotify.NewWatcher()
watcher.Add("config/config.yaml")
for event := range watcher.Events {
    if event.Op&fsnotify.Write == fsnotify.Write {
        // debounce then reload
    }
}
```

---

### 9. joho/godotenv

**Purpose:** Load `.env` file ke environment variables saat startup

**Note:** Hanya digunakan di development. Di production, env vars di-set langsung via Docker Compose atau systemd.

```go
godotenv.Load() // warn-only if .env doesn't exist
```

---

### 10. Logrus

**Purpose:** Structured logging

**Why Logrus?**
- **Structured:** JSON output untuk log aggregation
- **Fields:** Tambah context (category, insight_id)
- **Levels:** Debug, Info, Warn, Error, Fatal
- **Formatters:** JSON (production), Text (development)

**Usage:**
```go
import "github.com/sirupsen/logrus"

log.WithFields(logrus.Fields{
    "category":   "Golang",
    "insight_id": 42,
}).Info("sending insight")
```

---

### 11. Docker & Docker Compose

**Purpose:** Containerization dan deployment

**Key Features Used:**
- Multi-stage build untuk minimal image size (~10MB)
- Non-root user (`spark`) untuk security
- `restart: unless-stopped` untuk auto-restart
- Health check (`wget -q --spider http://localhost:8080/healthz`)
- Environment variables dari `.env` via `env_file`

**Dockerfile highlights:**
```dockerfile
FROM golang:1.23-alpine AS builder
# ...
FROM alpine:latest
RUN addgroup -g 1001 -S spark && adduser -u 1001 -S spark -G spark
USER spark
CMD ["./spark"]
```

---

### 12. lib/pq

**Purpose:** PostgreSQL driver untuk `database/sql`

**Why lib/pq?**
- **Pure Go:** Tidak ada C dependencies
- **Mature:** Stable, widely used
- **Array Support:** `pq.StringArray` untuk `TEXT[]` columns

**Usage:**
```go
import (
    _ "github.com/lib/pq"        // driver registration
    "github.com/lib/pq"
)

type Insight struct {
    Tags      pq.StringArray `db:"tags"`
    KeyPoints pq.StringArray `db:"key_points"`
}
```

---

## Technology Stack Summary

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|------|
| Language | Go | 1.23 | Core application |
| Database | PostgreSQL (Supabase) | 15+ | Data storage |
| Scheduler | robfig/cron | v3 | Job scheduling |
| Telegram | go-telegram-bot-api | v4.6.4 | Bot API client |
| LLM | OpenRouter (direct HTTP) | — | Content generation |
| Database Driver | sqlx + lib/pq | v1.4.0 / v1.12.3 | Database access |
| Config Parser | gopkg.in/yaml.v3 | v3.0.1 | YAML config |
| File Watcher | fsnotify | v1.10.1 | Config hot-reload |
| Env Loader | joho/godotenv | v1.5.1 | `.env` loading |
| Logging | Logrus | v1.9.4 | Structured logging |
| Deployment | Docker + Compose | Latest | Containerization |

---

## Actual go.mod Dependencies

```go
module github.com/navisha/spark

go 1.23.0

require (
    github.com/fsnotify/fsnotify v1.10.1
    github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
    github.com/jmoiron/sqlx v1.4.0
    github.com/lib/pq v1.12.3
    github.com/robfig/cron/v3 v3.0.1
    github.com/sirupsen/logrus v1.9.4
    gopkg.in/yaml.v3 v3.0.1
)

require (
    github.com/joho/godotenv v1.5.1 // indirect
    github.com/technoweenie/multipartstreamer v1.0.1 // indirect
    golang.org/x/sys v0.29.0 // indirect
)
```

---

## Environment Requirements

### Development
- Go 1.23+
- PostgreSQL 15+ (or Supabase account)
- Telegram Bot Token
- OpenRouter API Key

### Production (VPS)
- Docker 24+
- Docker Compose v2+
- 512MB RAM (minimum), 1GB recommended
- 1 vCPU
- 5GB disk

---

## Future Technology Considerations

### v2.0
- **Redis:** Untuk caching frequent queries
- **Prometheus + Grafana:** Untuk metrics dan monitoring

### v3.0
- **gRPC:** Untuk internal service communication jika multi-service
- **Kubernetes:** Jika perlu scale horizontally

---

**Document End**
