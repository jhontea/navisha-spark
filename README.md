# Navisha Spark 🔥

**Backend Engineering Learning Scheduler — Telegram Bot**

Navisha Spark adalah sistem pembelajaran backend engineering yang secara otomatis mengirimkan insight teknis ke Telegram setiap 3 jam. Dirancang untuk membantu senior backend engineer menjaga dan mempertajam pemahaman di berbagai topik inti melalui repetisi terjadwal, tanpa harus actively membuka materi belajar setiap hari.

---

## ✨ Fitur

| Fitur | Deskripsi |
|-------|-----------|
| ⏰ **Scheduled Delivery** | Kirim 1 insight teknis tiap 3 jam (00:00, 03:00, ..., 21:00 WIB) |
| 📚 **14 Topic Categories** | Golang, Database, System Design, Security, Kafka, Redis, Software Architecture, dll. |
| 🎯 **3 Difficulty Levels** | Beginner (20%), Intermediate (50%), Advanced (30%) |
| 🔄 **Smart Rotation** | Weighted round-robin + spaced repetition heuristic |
| 🚫 **Deduplication** | Tidak ada konten yang sama dalam 24 jam |
| 🔁 **Auto Retry** | Exponential backoff (1m, 5m, 15m) jika gagal |
| 🤖 **LLM Integration** | Generate insight on-the-fly via OpenRouter (free model) |
| 🔑 **Key-Based Topics** | Setiap insight memiliki key unik untuk prompt variation yang lebih baik |
| 🔧 **Hot-Reload Config** | Edit kategori/jadwal tanpa restart |
| 🐳 **Docker Ready** | Multi-stage build, health check, auto-restart |
| 🔒 **Chat ID Whitelist** | Hanya user tertentu yang bisa menerima pesan |

---

## 🏗️ Arsitektur

```
┌─────────────────────────────────────────────┐
│              Navisha Spark (Go)              │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │Scheduler │ │ Rotation │ │   Telegram   │ │
│  │(robfig/  │ │  Engine  │ │   Delivery   │ │
│  │  cron)   │ │          │ │              │ │
│  └────┬─────┘ └────┬─────┘ └──────┬───────┘ │
│       │            │              │         │
│       └────────────┼──────────────┘         │
│                    │                        │
│  ┌─────────────────┴──────────────────┐    │
│  │        Content Bank (PostgreSQL)    │    │
│  │  insights │ delivery_log │ rotation │    │
│  └────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
         │                    │
         ▼                    ▼
  ┌────────────┐     ┌──────────────┐
  │  Supabase  │     │   Telegram   │
  │ (Postgres) │     │   Bot API    │
  └────────────┘     └──────────────┘
```

---

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Supabase account (free tier)
- Telegram Bot Token (dari [@BotFather](https://t.me/BotFather))
- OpenRouter API Key (dari [openrouter.ai](https://openrouter.ai/))

### 1 Menit

```bash
# Clone
git clone https://github.com/yourusername/navisha-spark.git
cd navisha-spark

# Setup environment
cp .env.example .env
nano .env   # Isi kredensial

# Setup database (via Supabase SQL Editor)
# Buka https://supabase.com → SQL Editor → paste migrations/001_init.sql → Run

# Start
docker-compose up -d

# Cek health
curl http://localhost:8080/healthz
```

> 📖 **Dokumentasi lengkap:** [docs/SETUP.md](docs/SETUP.md)

---

## 📋 Topik yang Dicakup

| # | Kategori | Level |
|---|----------|-------|
| 1 | **Golang** | Beginner / Intermediate / Advanced |
| 2 | **Data Structures & Algorithms** | Beginner / Intermediate / Advanced |
| 3 | **Coding Challenge** | Beginner / Intermediate / Advanced |
| 4 | **Database** | Beginner / Intermediate / Advanced |
| 5 | **System Design** | Beginner / Intermediate / Advanced |
| 6 | **API Design** | Beginner / Intermediate / Advanced |
| 7 | **Deployment / DevOps** | Beginner / Intermediate / Advanced |
| 8 | **Security** | Beginner / Intermediate / Advanced |
| 9 | **Network** | Beginner / Intermediate / Advanced |
| 10 | **Caching (Redis)** | Beginner / Intermediate / Advanced |
| 11 | **Message Broker (Kafka)** | Beginner / Intermediate / Advanced |
| 12 | **Distributed Systems** | Beginner / Intermediate / Advanced |
| 13 | **Software Architecture** | Beginner / Intermediate / Advanced |
| 14 | **AI/ML untuk Backend Engineer** | Beginner / Intermediate / Advanced |

Kategori bisa ditambah/dikurangi lewat `config/config.yaml` tanpa restart.

---

## 📦 Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| **Language** | Go 1.23 |
| **Database** | PostgreSQL (Supabase) |
| **Scheduler** | robfig/cron v3 |
| **Telegram** | go-telegram-bot-api/v6 |
| **LLM** | OpenRouter (openrouter/owl-alpha) |
| **Config** | Viper + YAML (hot-reload) |
| **Logging** | Logrus (structured JSON) |
| **Database Driver** | sqlx + lib/pq |
| **Deployment** | Docker Compose |

---

## 📁 Project Structure

```
navisha-spark/
├── cmd/spark/main.go          # Entry point
├── internal/
│   ├── config/                # Configuration (env + YAML)
│   ├── database/              # PostgreSQL repository
│   ├── telegram/              # Telegram Bot API client
│   ├── content/               # LLM insight generation
│   ├── rotation/              # Topic selection & rotation
│   ├── scheduler/             # Cron scheduling
│   └── retry/                 # Exponential backoff
├── config/
│   └── config.yaml            # Unified config (categories, schedule, rotation, LLM, etc.)
├── migrations/                # Database migrations
├── agent/                     # AI assistant context & skills
├── docs/                      # Documentation
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

---

## ⚙️ Configuration

### Environment Variables (`.env`)

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `TELEGRAM_BOT_TOKEN` | ✅ | - | Bot token dari @BotFather |
| `TELEGRAM_CHAT_ID` | ✅ | - | Chat ID tujuan |
| `DATABASE_URL` | ✅ | - | Supabase connection string |
| `OPENROUTER_API_KEY` | ✅ | - | OpenRouter API key |
| `OPENROUTER_MODEL` | ❌ | `openrouter/owl-alpha` | Model LLM |
| `SCHEDULE_CRON` | ❌ | `0 */3 * * *` | Cron expression |
| `TIMEZONE` | ❌ | `Asia/Jakarta` | Timezone |
| `MAX_RETRIES` | ❌ | `3` | Max retry attempts |

### YAML Config (Hot-Reload)

- **`config/config.yaml`** — Satu file konfigurasi untuk semua: topik (categories), jadwal, level distribution, LLM, format, dan logging.

> Perubahan YAML langsung berlaku tanpa restart!

---

## 🧪 Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests
go test -tags=integration ./...
```

---

## 🐳 Deployment

### Docker Compose (Recommended)

```bash
docker-compose up -d --build
```

### Manual Binary

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o spark ./cmd/spark
./spark
```

> 📖 **Panduan deployment lengkap:** [docs/SETUP.md](docs/SETUP.md)

---

## 📊 Cost Estimate

| Komponen | Biaya/Bulan | Catatan |
|----------|:-----------:|---------|
| VPS (DigitalOcean) | $5 | 1GB RAM, 1 vCPU |
| Supabase (Free) | $0 | 500MB database |
| OpenRouter (Free) | $0-5 | Free model tersedia |
| **Total** | **$5-10** | Dalam target budget |

---

## 🔑 Key Column Feature

### Apa itu Key?

Key adalah identifier unik untuk setiap topik/subtopic dalam suatu kategori. Key digunakan untuk:

1. **Prompt Variation yang Lebih Baik**: LLM mendapatkan konteks yang lebih spesifik tentang topik yang ingin dibahas
2. **Deduplication yang Lebih Akurat**: Memastikan tidak ada insight dengan topik yang sama dalam jendela waktu tertentu
3. **Organisasi Konten yang Lebih Baik**: Memudahkan dalam mengelola dan mencari insight berdasarkan topik

### Format Key

Key menggunakan format `kebab-case` yang deskriptif, contoh:
- `goroutine-basics`
- `transaction-isolation`
- `cap-theorem`
- `redis-caching-strategies`
- `api-rate-limiting`

### Contoh Penggunaan

**Sebelum (tanpa key):**
```sql
INSERT INTO insights (category, level, title, insight, key_points, ...)
VALUES (
    'Golang',
    'beginner',
    'Understanding Goroutine',
    'Goroutine adalah thread ringan...',
    ARRAY['...'],
    ...
);
```

**Sesudah (dengan key):**
```sql
INSERT INTO insights (category, level, title, insight, key, key_points, ...)
VALUES (
    'Golang',
    'beginner',
    'Understanding Goroutine',
    'Goroutine adalah thread ringan...',
    'goroutine-basics',  -- Key baru
    ARRAY['...'],
    ...
);
```

### Manfaat untuk LLM Generation

Saat generate insight baru, sistem akan menggunakan key sebagai fokus prompt:

```
Buatkan insight pembelajaran level beginner tentang topik Golang dengan fokus pada: goroutine-basics dalam bahasa Indonesia.
```

Ini menghasilkan konten yang lebih terarah dan konsisten dibandingkan hanya menggunakan kategori umum.

### Unique Constraint

Database memiliki unique constraint pada `(category, key)` untuk memastikan tidak ada duplikasi topik dalam satu kategori:

```sql
CREATE UNIQUE INDEX idx_insights_category_key 
ON insights(category, key) 
WHERE key IS NOT NULL;
```

---

## 🗺️ Roadmap

### v1.0 (Sekarang)
- ✅ Scheduled delivery tiap 3 jam
- ✅ 14 kategori topik backend
- ✅ Smart rotation + deduplication
- ✅ LLM content generation
- ✅ Docker deployment

### v2.0 (Rencana)
- 🔲 Multi-user support
- 🔲 Web dashboard
- 🔲 Analytics & learning streak
- 🔲 Export to Anki/JSON

### v3.0 (Masa Depan)
- 🔲 Full SRS algorithm (SM-2)
- 🔲 Image/diagram support
- 🔲 Voice notes
- 🔲 Integration dengan Notion/Obsidian

---

## 🤝 Kontribusi

1. Fork repository
2. Buat branch: `git checkout -b feature/amazing-feature`
3. Commit: `git commit -m 'Add amazing feature'`
4. Push: `git push origin feature/amazing-feature`
5. Open Pull Request

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🙏 Credit

Dibuat dengan ❤️ untuk senior backend engineer yang ingin terus belajar.

> "Belajar adalah kebiasaan, bukan tugas."