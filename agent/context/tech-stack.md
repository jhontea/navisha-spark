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
- **Concurrency:** Built-in goroutine dan channel untuk concurrent programming
- **Standard Library:** Library yang lengkap (HTTP, JSON, SQL, dll.)
- **Single Binary:** Compile ke single binary, mudah di-deploy
- **Cross-platform:** Compile untuk berbagai OS dan architecture

**Key Features Used:**
- Goroutine untuk concurrent operations
- Channels untuk communication antar goroutine
- `context` package untuk cancellation dan timeout
- `net/http` untuk HTTP server (health check)
- `database/sql` + `sqlx` untuk database access

**Alternatives Considered:**
- Python: Lebih lambat, butuh GIL untuk concurrency
- Node.js: Single-threaded, tidak cocok untuk CPU-bound tasks
- Rust: Learning curve tinggi, overkill untuk project ini

---

### 2. PostgreSQL (Supabase)

**Purpose:** Primary database untuk menyimpan questions, delivery logs, dan rotation state

**Why PostgreSQL?**
- **ACID Compliant:** Jamin konsistensi data
- **JSONB Support:** Simpan follow_ups sebagai JSON dengan query yang efisien
- **Array Support:** Simpan tags sebagai array PostgreSQL
- **Reliable:** Battle-tested, mature, dan stabil
- **Managed (Supabase):** Tidak perlu manage database server sendiri
- **Free Tier:** Cukup untuk single user (500MB database)

**Key Features Used:**
- `SERIAL` untuk auto-increment primary key
- `JSONB` untuk flexible schema (follow_ups)
- `TEXT[]` untuk array of strings (tags)
- `CHECK` constraints untuk validasi (level enum)
- `TIMESTAMP DEFAULT NOW()` untuk audit fields
- `ON DELETE CASCADE` untuk referential integrity
- Indexes untuk query optimization

**Alternatives Considered:**
- MySQL: Tidak support JSONB dan array sebaik PostgreSQL
- SQLite: Tidak cocok untuk concurrent writes
- MongoDB: Overkill, tidak butuh document database
- Supabase vs self-hosted: Supabase lebih mudah, managed backup

**Connection:**
```
url supabase
```

---

### 3. robfig/cron v3

**Purpose:** In-process scheduler untuk trigger content delivery setiap 3 jam

**Why robfig/cron?**
- **Standard Library:** Paling populer untuk cron di Go
- **Flexible:** Support standard cron expressions
- **Lightweight:** Tidak ada dependency berat
- **In-process:** Tidak perlu external process atau OS-level timer
- **Timezone Support:** Bisa set timezone (Asia/Jakarta)

**Key Features Used:**
- `cron.New()` untuk create scheduler
- `cron.AddFunc()` untuk add job
- `cron.Start()` untuk start scheduler
- Timezone support via `cron.WithLocation()`

**Alternatives Considered:**
- systemd timer: Butuh akses host OS, tidak cocok untuk Docker
- Kubernetes CronJob: Overkill untuk single VPS
- go-cron: Kurva belajar lebih tinggi, dokumentasi kurang baik

**Usage:**
```go
import "github.com/robfig/cron/v3"

c := cron.New(cron.WithLocation(time.FixedLocation("WIB", 7*60*60)))
c.AddFunc("0 */3 * * *", sendQuestionJob)
c.Start()
```

---

### 4. go-telegram-bot-api/v6

**Purpose:** Official Telegram Bot API wrapper untuk Go

**Why go-telegram-bot-api?**
- **Official:** Maintained oleh Telegram team
- **Complete:** Support semua Bot API methods
- **Type-safe:** Strong typing untuk request dan response
- **Easy to Use:** API yang intuitive

**Key Features Used:**
- `telegram.NewBotAPI()` untuk create bot client
- `botAPI.Send()` untuk send message
- `telegram.NewMessage()` untuk create message
- `MessageConfig` untuk configure message

**Alternatives Considered:**
- Raw HTTP calls: Lebih ribet, manual parsing JSON
- grammY (TypeScript): Bukan Go
- python-telegram-bot: Bukan Go

**Usage:**
```go
import "github.com/go-telegram-bot-api/telegram-bot-api/v6"

bot, err := tgbotapi.NewBotAPI(token)
msg := tgbotapi.NewMessage(chatID, text)
msg.ParseMode = "Markdown"
bot.Send(msg)
```

---

### 5. OpenRouter

**Purpose:** LLM provider untuk generate questions on-the-fly

**Why OpenRouter?**
- **Unified API:** Single API untuk multiple LLM providers
- **Free Models:** Access ke free models (Mistral, Llama, Phi-3)
- **Easy to Switch:** Ganti model tanpa ubah code
- **Cost-effective:** Free tier untuk usage kecil

**Key Features Used:**
- Chat Completions API
- Model: `openrouter/owl-alpha`
- Streaming (optional, untuk future)

**Alternatives Considered:**
- OpenAI API: Berbayar, tidak ada free tier
- Anthropic Claude: Berbayar
- Local LLM (Ollama): Butuh GPU, resource-intensive
- Hugging Face Inference API: Rate limit ketat

**Usage:**
```go
import "github.com/openai/openai-go"

client := openai.NewClient(apiKey)
resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Model: openai.String("openrouter/owl-alpha"),
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage(prompt),
    },
})
```

---

### 6. sqlx

**Purpose:** Extension untuk `database/sql` dengan fitur-fitur tambahan

**Why sqlx?**
- **Type-safe:** Auto-map rows ke struct
- **Named Queries:** Support named parameters (`:name`)
- **Struct Tags:** Mapping dengan `db:"column_name"`
- **Lightweight:** Tidak seperti ORM (GORM), tetap kontrol penuh atas SQL
- **Performance:** Tidak ada overhead ORM

**Key Features Used:**
- `db.Get()` untuk single row
- `db.Select()` untuk multiple rows
- `db.NamedQuery()` untuk named parameters
- `db.MustExec()` untuk exec tanpa error handling

**Alternatives Considered:**
- GORM: Terlalu magic, kontrol kurang, learning curve tinggi
- Raw `database/sql`: Manual scanning, verbose
- pgx: Lebih cepat tapi API kurang friendly
- ent: Overkill untuk project ini

**Usage:**
```go
import "github.com/jmoiron/sqlx"

var q Question
err := db.Get(&q, "SELECT * FROM questions WHERE id = $1", id)
```

---

### 7. Viper

**Purpose:** Configuration management (env vars + YAML files)

**Why Viper?**
- **Multiple Sources:** Env vars, YAML, JSON, TOML
- **Hot-reload:** Bisa reload config tanpa restart
- **Default Values:** Support default values
- **Widely Used:** Standard untuk config di Go ecosystem

**Key Features Used:**
- `viper.AutomaticEnv()` untuk bind env vars
- `viper.ReadInConfig()` untuk read YAML
- `viper.Unmarshal()` untuk parse ke struct
- `viper.WatchConfig()` untuk hot-reload (optional)

**Alternatives Considered:**
- envconfig: Hanya support env vars, tidak ada YAML
- koanf: Lebih baru, dokumentasi kurang
- godotenv: Hanya load `.env`, tidak ada advanced features

**Usage:**
```go
import "github.com/spf13/viper"

viper.SetConfigFile("config/schedule.yaml")
viper.AutomaticEnv()
viper.ReadInConfig()
viper.Unmarshal(&cfg)
```

---

### 8. Logrus

**Purpose:** Structured logging

**Why Logrus?**
- **Structured:** JSON output untuk log aggregation
- **Fields:** Bisa tambah fields (category, level, question_id)
- **Levels:** Debug, Info, Warn, Error, Fatal
- **Formatters:** JSON, Text, custom formatters

**Key Features Used:**
- `logrus.WithFields()` untuk structured logging
- `logrus.Info()`, `logrus.Error()` untuk log levels
- JSON formatter untuk production

**Alternatives Considered:**
- zap (Uber): Lebih cepat tapi API kurang intuitive
- zerolog: Lebih baru, dokumentasi kurang
- standard `log`: Tidak ada structured logging

**Usage:**
```go
import "github.com/sirupsen/logrus"

logrus.WithFields(logrus.Fields{
    "category": "Golang",
    "level": "intermediate",
}).Info("Sending question")
```

---

### 9. Docker & Docker Compose

**Purpose:** Containerization dan deployment

**Why Docker?**
- **Portability:** Run di anywhere (dev, staging, prod)
- **Isolation:** Dependencies terisolasi per container
- **Reproducibility:** Build once, run anywhere
- **Easy Deployment:** Single command untuk start/stop

**Why Docker Compose?**
- **Multi-container:** Define multiple services di satu file
- **Simple:** Tidak perlu Kubernetes untuk single service
- **Environment:** Define env vars, volumes, networks
- **Restart Policy:** Auto-restart on crash

**Key Features Used:**
- Multi-stage build untuk minimal image size
- `restart: unless-stopped` untuk auto-restart
- Health check untuk monitoring
- Environment variables dari `.env`

**Alternatives Considered:**
- Kubernetes: Overkill untuk single service
- systemd: Butuh akses host, kurang portable
- Bare metal: Sulit manage dependencies

---

### 10. lib/pq

**Purpose:** PostgreSQL driver untuk `database/sql`

**Why lib/pq?**
- **Pure Go:** Tidak ada C dependencies
- **Mature:** Stable, widely used
- **SSL Support:** Support SSL/TLS connections
- **Complete:** Support semua PostgreSQL features

**Key Features Used:**
- Connection string dengan SSL
- Array support (`tags TEXT[]`)
- JSONB support

**Alternatives Considered:**
- pgx: Lebih cepat tapi kurang compatible dengan `database/sql`
- go-pg: ORM-style, tidak cocok dengan sqlx

---

## Supporting Technologies

### 11. YAML

**Purpose:** Configuration files (categories, schedule)

**Why YAML?**
- **Human-readable:** Mudah dibaca dan diedit
- **Hierarchical:** Support nested structures
- **Widely Supported:** Banyak library di semua bahasa

**Libraries:**
- `gopkg.in/yaml.v3` (via Viper)

---

### 12. Markdown

**Purpose:** Message formatting untuk Telegram

**Why Markdown?**
- **Simple:** Easy to write dan read
- **Telegram Support:** Telegram Bot API support Markdown parsing
- **Formatting:** Bold, italic, code blocks, lists

**Usage:**
```markdown
📚 *Golang* — *Intermediate*

*Pertanyaan:*
Apa itu goroutine?

💡 *Jawaban:*
Goroutine adalah thread ringan...

🔍 *Follow-up:*
• Bagaimana cara membuat goroutine?
• Apa perbedaan goroutine dan thread?
```

---

## Technology Stack Summary

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| Language | Go | 1.23 | Core application |
| Database | PostgreSQL (Supabase) | 15+ | Data storage |
| Scheduler | robfig/cron | v3 | Job scheduling |
| Telegram | go-telegram-bot-api | v6 | Bot API client |
| LLM | OpenRouter | - | Content generation |
| Database Driver | sqlx + lib/pq | v1.4.0 / v1.10.9 | Database access |
| Config | Viper | v1.19.0 | Configuration management |
| Logging | Logrus | v1.9.3 | Structured logging |
| Deployment | Docker + Compose | Latest | Containerization |
| Config Format | YAML | v3 | Configuration files |
| Message Format | Markdown | - | Telegram messages |

---

## Dependency Tree

```
github.com/navisha/spark
├── github.com/go-telegram-bot-api/telegram-bot-api/v6
│   └── (no dependencies)
├── github.com/jmoiron/sqlx
│   └── github.com/lib/pq
│       └── (no dependencies)
├── github.com/robfig/cron/v3
│   └── (no dependencies)
├── github.com/sirupsen/logrus
│   └── github.com/sirupsen/logrus
├── github.com/spf13/viper
│   ├── github.com/fsnotify/fsnotify (optional, for hot-reload)
│   ├── github.com/spf13/cast
│   └── github.com/spf13/pflag
└── github.com/openai/openai-go (for OpenRouter)
    └── (HTTP client)
```

---

## Version Constraints

```go
// go.mod
module github.com/navisha/spark

go 1.23

require (
    github.com/go-telegram-bot-api/telegram-bot-api/v6 v6.1.0
    github.com/jmoiron/sqlx v1.4.0
    github.com/lib/pq v1.10.9
    github.com/robfig/cron/v3 v3.0.1
    github.com/sirupsen/logrus v1.9.3
    github.com/spf13/cast v1.6.0
    github.com/spf13/viper v1.19.0
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
- 512MB RAM (minimum)
- 1 vCPU
- 5GB disk

---

## Future Technology Considerations

### v2.0
- **Redis:** Untuk caching frequent queries
- **Prometheus + Grafana:** Untuk metrics dan monitoring
- **Kafka:** Untuk message queue (jika scale ke multi-user)

### v3.0
- **gRPC:** Untuk internal service communication
- **Kubernetes:** Jika perlu scale horizontally
- **Terraform:** Untuk infrastructure as code

---

**Document End**