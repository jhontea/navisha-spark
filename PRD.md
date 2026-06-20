# Navisha Spark — Product Requirements Document

**Version:** 1.0  
**Date:** 2025-06-20  
**Author:** Navisha Spark Team  
**Status:** Draft

---

## 1. Executive Summary

Navisha Spark adalah sistem pembelajaran backend engineering yang mengirimkan satu insight teknis ke Telegram setiap 3 jam secara otomatis. Sistem ini dirancang untuk membantu senior backend engineer menjaga dan meningkatkan pemahaman di berbagai topik inti melalui repetisi terjadwal (spaced repetition) tanpa intervensi manual.

### Key Metrics
- **Target Cost:** <$5-10/bulan (termasuk LLM API)
- **Delivery Frequency:** Tiap 3 jam (8x sehari)
- **Uptime Target:** 99.9% (auto-restart on crash)
- **Content Accuracy:** Prioritas bank soal kurasi, LLM hanya untuk variasi/follow-up

---

## 2. Problem Statement

Senior backend engineer yang mempersiapkan technical interview membutuhkan cara konsisten untuk mengingat dan mempertajam pemahaman di berbagai topik inti. Materi belajar sudah banyak tersedia, tetapi tidak ada mekanisme delivery otomatis yang mendorong repetisi terjadwal. Hasilnya, belajar menjadi tugas tambahan yang mudah diabaikan.

### Pain Points
1. **Inconsistent Learning:** Tidak ada pengingat terjadwal untuk review materi
2. **Context Switching:** Harus actively membuka materi belajar setiap hari
3. **Knowledge Decay:** Topik yang tidak sering dipakai cepat terlupakan
4. **No Progress Tracking:** Sulit melacak konsistensi pembelajaran

---

## 3. Goals & Objectives

### Primary Goals
1. **Automated Delivery:** Mengirim 1 insight teknis ke Telegram setiap 3 jam tanpa intervensi manual
2. **Balanced Coverage:** Rotasi merata antar 13 kategori topik backend engineering
3. **Technical Accuracy:** Konten akurat secara teknis, dengan LLM hanya untuk variasi
4. **Cost Efficiency:** Sistem ringan, aman, dan murah (<$5-10/bulan)

### Success Criteria
- ✅ Konten terkirim pada jam 00:00, 03:00, 06:00, 09:00, 12:00, 15:00, 18:00, 21:00 WIB
- ✅ Retry maksimal 3x dengan exponential backoff (1m, 5m, 15m) jika gagal
- ✅ Log error dan skip ke jadwal berikutnya jika semua retry gagal
- ✅ Tidak ada duplicate content dalam 24 jam
- ✅ Sistem berjalan terus menerus (auto-restart on crash)

---

## 4. System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        VPS (Docker)                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Navisha Spark (Go Binary)                 │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐ │  │
│  │  │   Scheduler │  │   Rotation   │  │   Telegram   │ │  │
│  │  │  (robfig/   │  │   Engine     │  │   Delivery   │ │  │
│  │  │   cron)     │  │              │  │              │ │  │
│  │  └──────┬──────┘  └──────┬───────┘  └──────┬───────┘ │  │
│  │         │                │                  │         │  │
│  │         └────────────────┼──────────────────┘         │  │
│  │                          │                            │  │
│  │  ┌───────────────────────┴───────────────────────┐   │  │
│  │  │           Content Bank (PostgreSQL)            │   │  │
│  │  │  ┌──────────────┐  ┌──────────────────────┐  │   │  │
│  │  │  │   questions  │  │   rotation_state     │  │   │  │
│  │  │  └──────────────┘  └──────────────────────┘  │   │  │
│  │  │  ┌──────────────┐  ┌──────────────────────┐  │   │  │
│  │  │  │ delivery_log │  │   sent_history       │  │   │  │
│  │  │  └──────────────┘  └──────────────────────┘  │   │  │
│  │  └───────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          │ HTTPS                             │
│                          ▼                                   │
│              ┌─────────────────────┐                        │
│              │   Supabase (Postgres)│                        │
│              │   aws-1-ap-southeast-1│                       │
│              │   .pooler.supabase.com│                       │
│              └─────────────────────┘                        │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS
                          ▼
              ┌─────────────────────┐
              │   Telegram Bot API  │
              │   (sendMessage)     │
              └─────────────────────┘
                          │
                          ▼
              ┌─────────────────────┐
              │   User Telegram     │
              │   (chat_id: 203294061)│
              └─────────────────────┘
```

### Component Interaction

1. **Scheduler** (robfig/cron) — Trigger setiap 3 jam sesuai timezone WIB
2. **Rotation Engine** — Pilih kategori dan level berdasarkan weighted round-robin + heuristik spaced repetition
3. **Content Bank** — Ambil soal dari PostgreSQL, generate via LLM jika kosong
4. **Telegram Delivery** — Kirim pesan dengan format Markdown, retry jika gagal
5. **Database** — Simpan state rotasi, log delivery, dan history pengiriman

---

## 5. Functional Requirements

### 5.1 Topic Taxonomy

Bank soal dikelompokkan ke dalam 13 kategori, masing-masing dengan 3 level kesulitan:

| # | Kategori | Contoh Subtopik |
|---|----------|-----------------|
| 1 | Golang | concurrency, goroutine/channel, GMP scheduler, slice, generics, profiling |
| 2 | Data Structures & Algorithms | array/tree/graph, dynamic programming, complexity analysis |
| 3 | Coding Challenge | worker pool, pipeline, circuit breaker, rate limiter (Go-based) |
| 4 | Database | indexing, query planning, replication, transaction isolation |
| 5 | System Design | CAP theorem, consistent hashing, saga/2PC, idempotency |
| 6 | API Design | REST vs GraphQL, versioning, rate limiting, gateway/BFF |
| 7 | Deployment / DevOps | CI/CD, Kubernetes, GitOps, multi-region, disaster recovery |
| 8 | Security | authn/authz, JWT, CSRF, IDOR, OAuth 2.0, secrets management |
| 9 | Network | TCP/IP, HTTP internals, DNS, load balancing, TLS handshake |
| 10 | Caching (Redis) | caching strategy, distributed lock, cluster, pub/sub vs streams |
| 11 | Message Broker (Kafka) | partisi, consumer group, exactly-once, KRaft |
| 12 | Distributed Systems | consensus, replication, partition tolerance, failure modes |
| 13 | AI/ML untuk Backend Engineer | dasar LLM, vector DB, RAG, ML system design dari sudut pandang backend |

**Requirement:** Kategori bisa ditambah/dikurangi lewat config (`config/categories.yaml`) tanpa redeploy.

### 5.2 Content Bank (Insight Store)

Setiap entri berisi:
- `category` — Kategori topik
- `level` — beginner / intermediate / advanced
- `title` — Judul insight/topic
- `insight` — Insight lengkap dan detail yang bisa dipelajari (bukan hanya Q&A)
- `key_points` — Array poin-poin penting (bullet points)
- `code_example` — Contoh kode atau konfigurasi (opsional, untuk kategori teknis)
- `follow_ups` — Array follow-up questions beserta jawabannya (opsional)
- `tags` — Array tag/topik spesifik
- `times_sent` — Counter pengiriman
- `last_sent_at` — Timestamp pengiriman terakhir
- `created_at` — Timestamp pembuatan
- `updated_at` — Timestamp update terakhir

**Sumber Konten:**
1. **Primary:** Bank insight kurasi (pre-seeded atau di-generate via LLM)
2. **Fallback:** LLM generation on-the-fly (OpenRouter) jika bank kosong

**Content Guidelines:**
- Setiap insight harus **lengkap dan detail** (minimal 300 kata)
- Sertakan **contoh praktis** dan **code snippets** jika relevan
- Jelaskan **kapan menggunakan** dan **kapan tidak menggunakan** konsep tersebut
- Sertakan **common pitfalls** dan **best practices**
- Follow-up questions harus **disertai jawaban** untuk pembelajaran yang komprehensif

### 5.3 Scheduling Engine

**Trigger:** Cron expression `0 */3 * * *` (tiap 3 jam)  
**Timezone:** Asia/Jakarta (WIB)  
**Active Hours:** Konfigurasi via `config/schedule.yaml` (default 00:00-23:00)

**Implementation:** robfig/cron v3 — in-process scheduler dalam Go binary

**Why in-process:**
- Go binary sangat ringan (~10-15MB RAM, ~0.1% CPU saat idle)
- State tersimpan di PostgreSQL — tidak hilang saat restart
- Docker restart policy (`unless-stopped`) cukup untuk auto-restart
- Lebih simpel untuk VPS single-node

### 5.4 Topic Selection & Rotation Logic

#### Weighted Round-Robin Antar Kategori
- Kategori yang paling lama tidak dikirim mendapat prioritas lebih tinggi
- Formula: `priority = (now - last_sent_at) * weight`
- Weight default: 1.0 untuk semua kategori (bisa dikonfigurasi)

#### Level Distribution (Spaced Repetition Heuristic)
- Rasio level: **20% beginner, 50% intermediate, 30% advanced**
- Jangan ulang soal yang sama dalam `MIN_DAYS_BEFORE_REPEAT` (default 7 hari)
- Naikkan level jika user sudah menguasai level sebelumnya (berdasarkan `times_sent`)

#### Deduplication
- Tidak kirim soal yang sama dalam `DEDUP_WINDOW_HOURS` (default 24 jam)
- Cek `sent_history` table sebelum memilih soal

### 5.5 Telegram Delivery

**API:** Telegram Bot API — `sendMessage`  
**Format:** Markdown (kategori, level, pertanyaan jelas terpisah)  
**Chat ID:** Whitelist hanya untuk user sendiri (`203294061`)

**Message Format:**
```markdown
📚 *[Kategori]* — *[Level]*

*[Title]*

📝 *Insight:*
[Insight lengkap dan detail dengan contoh praktis]

💡 *Key Points:*
• Point 1
• Point 2
• Point 3

🔍 *Deep Dive:*
*Q: [Follow-up question]*
A: [Jawaban follow-up]

*Q: [Follow-up question]*
A: [Jawaban follow-up]

---
_Tags: golang, concurrency, channel_
```

**Retry Logic:**
- Maksimal 3x retry dengan exponential backoff: 1 menit, 5 menit, 15 menit
- Jika semua retry gagal: log error dan skip ke jadwal berikutnya

### 5.6 LLM Integration (OpenRouter)

**Model:** `openrouter/owl-alpha`  
**Purpose:** Generate content on-the-fly jika bank soal kosong, atau generate follow-up questions

**Prompt Template:**
```
Generate a {level} level learning insight about {category} topic: {subtopic}.

Requirements:
1. Title: Clear and specific (max 100 chars)
2. Insight: Comprehensive explanation (300-500 words) with:
   - Definition and core concepts
   - Practical examples and use cases
   - Code snippets if applicable
   - When to use / when not to use
   - Common pitfalls and best practices
3. Key Points: 3-5 bullet points summarizing the insight
4. Follow-ups: 2-3 questions with detailed answers (each answer 100-200 words)
5. Tags: 3-5 relevant tags

Format as JSON:
{
    "title": "Understanding Goroutine",
    "insight": "Detailed explanation...",
    "key_points": ["Point 1", "Point 2", "Point 3"],
    "code_example": "func main() {...}",
    "follow_ups": [
        {"q": "Question 1?", "a": "Answer 1"},
        {"q": "Question 2?", "a": "Answer 2"}
    ],
    "tags": ["golang", "concurrency"]
}
```

**Fallback Strategy:**
1. Cari soal di bank yang belum dikirim dalam 24 jam
2. Jika tidak ada, generate via LLM
3. Simpan hasil generate ke bank untuk penggunaan selanjutnya

---

## 6. Non-Functional Requirements

### 6.1 Performance
- **Response Time:** <2 detik untuk Telegram API call
- **Database Query:** <500ms untuk query rotasi
- **Memory Usage:** <50MB saat idle
- **CPU Usage:** <5% saat idle

### 6.2 Reliability
- **Uptime:** 99.9% (Docker restart policy + health check)
- **Retry:** Exponential backoff dengan max 3x
- **Error Handling:** Log semua error ke stdout/stderr (Docker logging driver)

### 6.3 Security
- **Secrets Management:** Environment variables via `.env` (tidak di-commit)
- **Chat ID Whitelist:** Hanya accept message dari chat_id yang diizinkan
- **Database:** Connection via SSL (Supabase pooler)
- **LLM API:** Key via environment variable

### 6.4 Scalability
- **Horizontal:** Tidak diperlukan untuk single user
- **Vertical:** Go binary bisa handle 1000+ requests/day dengan mudah
- **Database:** PostgreSQL via Supabase — managed, auto-scaling

### 6.5 Maintainability
- **Code Structure:** Clean architecture (internal/ packages)
- **Configuration:** YAML files untuk categories dan schedule
- **Logging:** Structured logging (JSON) untuk monitoring
- **Documentation:** Agent context + skills untuk AI-assisted development

---

## 7. Database Schema

### 7.1 insights
```sql
CREATE TABLE insights (
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

CREATE INDEX idx_insights_category_level ON insights(category, level);
CREATE INDEX idx_insights_last_sent_at ON insights(last_sent_at);
CREATE INDEX idx_insights_tags ON insights USING GIN(tags);
```

### 7.2 delivery_log
```sql
CREATE TABLE delivery_log (
    id SERIAL PRIMARY KEY,
    insight_id INT REFERENCES insights(id),
    sent_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success','failed','retry')),
    error_message TEXT,
    telegram_message_id BIGINT
);

CREATE INDEX idx_delivery_log_sent_at ON delivery_log(sent_at);
CREATE INDEX idx_delivery_log_insight_id ON delivery_log(insight_id);
```

### 7.3 rotation_state
```sql
CREATE TABLE rotation_state (
    category VARCHAR(100) PRIMARY KEY,
    last_sent_at TIMESTAMP,
    total_sent INT DEFAULT 0,
    last_level VARCHAR(20),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 7.4 sent_history
```sql
CREATE TABLE sent_history (
    insight_id INT REFERENCES insights(id),
    sent_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (insight_id, sent_at)
);

CREATE INDEX idx_sent_history_sent_at ON sent_history(sent_at);
CREATE INDEX idx_sent_history_insight_id ON sent_history(insight_id);
```

---

## 8. Configuration

### 8.1 Environment Variables
Lihat `.env.example` untuk daftar lengkap.

### 8.2 config/categories.yaml
```yaml
categories:
  - name: "Golang"
    enabled: true
    weight: 1.0
    subtopics:
      - "concurrency"
      - "goroutine/channel"
      - "GMP scheduler"
      - "slice"
      - "generics"
      - "profiling"
  
  - name: "Data Structures & Algorithms"
    enabled: true
    weight: 1.0
    subtopics:
      - "array/tree/graph"
      - "dynamic programming"
      - "complexity analysis"
  
  # ... 11 kategori lainnya
```

### 8.3 config/schedule.yaml
```yaml
schedule:
  cron: "0 */3 * * *"
  timezone: "Asia/Jakarta"
  active_hours:
    start: 0
    end: 23

rotation:
  level_distribution:
    beginner: 20
    intermediate: 50
    advanced: 30
  min_days_before_repeat: 7

deduplication:
  window_hours: 24

retry:
  max_retries: 3
  delays:
    - 1m
    - 5m
    - 15m
```

---

## 9. Deployment Architecture

### 9.1 Docker Compose
```yaml
services:
  spark:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
      - OPENROUTER_MODEL=${OPENROUTER_MODEL}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3
```

### 9.2 Dockerfile
```dockerfile
# Multi-stage build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o spark ./cmd/spark

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/spark .
COPY --from=builder /app/config/ ./config/
EXPOSE 8080
CMD ["./spark"]
```

### 9.3 VPS Requirements
- **OS:** Ubuntu 22.04 / Debian 12
- **RAM:** 512MB (minimum), 1GB (recommended)
- **CPU:** 1 vCPU
- **Disk:** 5GB
- **Cost:** ~$5/month (DigitalOcean/Vultr) + ~$0-5/month (OpenRouter free tier)

---

## 10. Monitoring & Observability

### 10.1 Health Check
- **Endpoint:** `GET /healthz`
- **Response:** `{"status": "ok", "timestamp": "2025-06-20T21:00:00Z"}`

### 10.2 Metrics
- **Endpoint:** `GET /metrics` (Prometheus format, optional)
- **Metrics:**
  - `spark_messages_sent_total` — Total pesan terkirim
  - `spark_messages_failed_total` — Total pesan gagal
  - `spark_questions_generated_total` — Total soal di-generate
  - `spark_rotation_duration_seconds` — Durasi proses rotasi

### 10.3 Logging
- **Format:** Structured JSON
- **Fields:** timestamp, level, message, category, question_id, error
- **Output:** stdout/stderr (Docker logging driver)

---

## 11. Cost Estimate

| Component | Cost/Month | Notes |
|-----------|-----------|-------|
| VPS (DigitalOcean Droplet) | $5 | 1GB RAM, 1 vCPU, 25GB SSD |
| Supabase (Free Tier) | $0 | 500MB database, 2GB bandwidth |
| OpenRouter (Free Tier) | $0-5 | openrouter/owl-alpha (free model) |
| **Total** | **$5-10** | Within target budget |

---

## 12. Future Enhancements

### v2.0
- Multi-user support (whitelist multiple chat_id)
- Web dashboard untuk manage questions
- Analytics (learning streak, topic mastery)
- Export questions to Anki/JSON

### v3.0
- Full SRS algorithm (SM-2)
- Image/diagram support untuk questions
- Voice notes (Telegram voice message)
- Integration dengan Notion/Obsidian

---

## 13. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM generate inaccurate content | High | Prioritaskan bank soal kurasi, LLM hanya untuk variasi |
| Supabase downtime | Medium | Retry logic + error logging, skip ke jadwal berikutnya |
| Telegram API rate limit | Low | 8x/hari << 30 msg/min limit |
| Cost overrun | Medium | Monitor LLM API usage, switch ke free model jika perlu |
| Docker container crash | Low | `restart: unless-stopped` + health check |

---

## 14. Appendix

### 14.1 Glossary
- **SRS (Spaced Repetition System):** Algoritma untuk menentukan kapan materi harus di-review kembali
- **Weighted Round-Robin:** Algoritma rotasi dengan prioritas berbobot
- **Exponential Backoff:** Strategi retry dengan delay yang meningkat secara eksponensial
- **Deduplication:** Proses menghindari pengiriman konten yang sama berkali-kali

### 14.2 References
- [robfig/cron](https://github.com/robfig/cron)
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [OpenRouter](https://openrouter.ai/)
- [Supabase](https://supabase.com/)
- [sqlx](https://jmoiron.github.io/sqlx/)

---

**Document End**