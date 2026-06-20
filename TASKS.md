# Navisha Spark — Implementation Tasks

## Overview

Daftar task untuk implementasi Navisha Spark. Dibagi menjadi beberapa fase berdasarkan prioritas dan dependensi.

---

## Fase 1: Project Foundation (Setup)

### Task 1.1 — Initialize Go Module & Dependencies
- [x] `go mod tidy` untuk download semua dependencies
- [x] Buat `go.sum` (lock file)
- [x] Verifikasi semua package bisa di-import
- [x] Setup linter (golangci-lint) — optional

**Files:** `go.mod`, `go.sum`  
**Dependencies:** Semua package di go.mod  
**Estimasi:** 15 menit

### Task 1.2 — Database Migration di Supabase
- [x] Buka Supabase SQL Editor
- [x] Jalankan `migrations/001_init.sql`
- [x] Verifikasi 4 table terbuat: `insights`, `delivery_log`, `rotation_state`, `sent_history`
- [x] Test koneksi dari lokal: `psql $DATABASE_URL -c "SELECT 1"`

**Files:** `migrations/001_init.sql`  
**Dependencies:** Supabase project aktif  
**Estimasi:** 10 menit

### Task 1.3 — Seed Initial Data
- [x] Insert sample insights (3-5 baris) untuk testing
- [x] Insert rotation_state untuk 13 kategori
- [x] Verifikasi data dengan SELECT queries

**Files:** `migrations/001_init.sql` (bagian seed)  
**Dependencies:** Task 1.2  
**Estimasi:** 10 menit

---

## Fase 2: Core Library (internal/)

### Task 2.1 — Config Package
- [ ] Buat `internal/config/config.go` — struct definitions
- [ ] Buat `internal/config/loader.go` — load dari env + YAML
- [ ] Buat `internal/config/types.go` — type definitions
- [ ] Implementasi hot-reload dengan fsnotify
- [ ] Unit test untuk config loading

**Files:**
- `internal/config/config.go`
- `internal/config/loader.go`
- `internal/config/types.go`
- `internal/config/config_test.go`

**Dependencies:** Task 1.1  
**Estimasi:** 2 jam

### Task 2.2 — Database Package
- [ ] Buat `internal/database/connection.go` — koneksi pool
- [ ] Buat `internal/database/repository/insight.go` — CRUD insights
- [ ] Buat `internal/database/repository/delivery.go` — delivery log
- [ ] Buat `internal/database/repository/rotation.go` — rotation state
- [ ] Buat `internal/database/repository/history.go` — sent history
- [ ] Buat `internal/database/migration/runner.go` — migration runner
- [ ] Unit test dengan test database

**Files:**
- `internal/database/connection.go`
- `internal/database/repository/insight.go`
- `internal/database/repository/delivery.go`
- `internal/database/repository/rotation.go`
- `internal/database/repository/history.go`
- `internal/database/migration/runner.go`

**Dependencies:** Task 1.1, 1.2  
**Estimasi:** 4 jam

### Task 2.3 — Telegram Package
- [ ] Buat `internal/telegram/client.go` — bot client
- [ ] Buat `internal/telegram/formatter.go` — Markdown formatter untuk insight
- [ ] Buat `internal/telegram/whitelist.go` — chat ID validation
- [ ] Implementasi format pesan: title, insight, key_points, follow_ups (Q&A)
- [ ] Unit test dengan mock

**Files:**
- `internal/telegram/client.go`
- `internal/telegram/formatter.go`
- `internal/telegram/whitelist.go`
- `internal/telegram/client_test.go`

**Dependencies:** Task 1.1  
**Estimasi:** 3 jam

### Task 2.4 — Content Package (LLM)
- [ ] Buat `internal/content/generator.go` — LLM client (OpenRouter)
- [ ] Buat `internal/content/prompt.go` — prompt templates untuk insight generation
- [ ] Buat `internal/content/validator.go` — validasi hasil generate
- [ ] Implementasi JSON parsing dengan fallback
- [ ] Unit test dengan mock LLM

**Files:**
- `internal/content/generator.go`
- `internal/content/prompt.go`
- `internal/content/validator.go`
- `internal/content/generator_test.go`

**Dependencies:** Task 1.1  
**Estimasi:** 3 jam

### Task 2.5 — Rotation Package
- [ ] Buat `internal/rotation/engine.go` — main rotation logic
- [ ] Buat `internal/rotation/selector.go` — weighted round-robin
- [ ] Buat `internal/rotation/dedup.go` — deduplication logic
- [ ] Buat `internal/rotation/spaced_repetition.go` — level heuristics
- [ ] Unit test untuk setiap komponen

**Files:**
- `internal/rotation/engine.go`
- `internal/rotation/selector.go`
- `internal/rotation/dedup.go`
- `internal/rotation/spaced_repetition.go`
- `internal/rotation/engine_test.go`

**Dependencies:** Task 2.2  
**Estimasi:** 4 jam

### Task 2.6 — Scheduler Package
- [ ] Buat `internal/scheduler/scheduler.go` — cron wrapper
- [ ] Buat `internal/scheduler/job.go` — job definition (send insight)
- [ ] Buat `internal/scheduler/timezone.go` — WIB timezone handling
- [ ] Implementasi active hours check
- [ ] Unit test

**Files:**
- `internal/scheduler/scheduler.go`
- `internal/scheduler/job.go`
- `internal/scheduler/timezone.go`
- `internal/scheduler/scheduler_test.go`

**Dependencies:** Task 2.3, 2.4, 2.5  
**Estimasi:** 2 jam

### Task 2.7 — Retry Package
- [ ] Buat `internal/retry/retry.go` — retry logic
- [ ] Buat `internal/retry/backoff.go` — exponential backoff
- [ ] Buat `internal/retry/policy.go` — retry policy
- [ ] Unit test

**Files:**
- `internal/retry/retry.go`
- `internal/retry/backoff.go`
- `internal/retry/policy.go`
- `internal/retry/retry_test.go`

**Dependencies:** Task 1.1  
**Estimasi:** 1 jam

---

## Fase 3: Application Entry Point

### Task 3.1 — Main Application
- [ ] Buat `cmd/spark/main.go` — entry point
- [ ] Inisialisasi semua service
- [ ] Setup graceful shutdown (SIGINT/SIGTERM)
- [ ] Setup health check endpoint (`/healthz`)
- [ ] Setup scheduler
- [ ] Logging startup

**Files:**
- `cmd/spark/main.go`

**Dependencies:** Semua Fase 2  
**Estimasi:** 3 jam

### Task 3.2 — HTTP Server (Health Check)
- [ ] Buat HTTP server untuk health check
- [ ] Implementasi endpoint `GET /healthz`
- [ ] Cek database connection
- [ ] Cek Telegram API reachability
- [ ] Return JSON response

**Files:** (bagian dari `cmd/spark/main.go` atau `internal/http/server.go`)  
**Dependencies:** Task 3.1  
**Estimasi:** 1 jam

---

## Fase 4: Integration & Testing

### Task 4.1 — Integration Test
- [ ] Test end-to-end: scheduler trigger → rotation → content → telegram
- [ ] Test retry logic dengan simulated failure
- [ ] Test deduplication (kirim insight yang sama dalam 24 jam)
- [ ] Test hot-reload config
- [ ] Test graceful shutdown

**Files:** `*_test.go` di setiap package  
**Dependencies:** Semua Fase 3  
**Estimasi:** 3 jam

### Task 4.2 — Manual Testing via Telegram
- [ ] Start aplikasi
- [ ] Tunggu jadwal pertama (atau trigger manual)
- [ ] Verifikasi format pesan di Telegram
- [ ] Verifikasi follow-up questions muncul dengan jawaban
- [ ] Verifikasi key points terformat dengan baik
- [ ] Verifikasi tidak ada duplicate content

**Dependencies:** Task 4.1  
**Estimasi:** 2 jam (menunggu jadwal)

---

## Fase 5: Deployment

### Task 5.1 — Docker Build & Test
- [ ] Build Docker image: `docker-compose build`
- [ ] Test run lokal: `docker-compose up`
- [ ] Verifikasi health check
- [ ] Verifikasi logs
- [ ] Optimasi image size

**Files:** `Dockerfile`, `docker-compose.yml`  
**Dependencies:** Semua Fase 3  
**Estimasi:** 1 jam

### Task 5.2 — Deploy ke VPS
- [ ] SSH ke VPS
- [ ] Install Docker + Docker Compose
- [ ] Clone repository
- [ ] Setup `.env` dengan kredensial asli
- [ ] `docker-compose up -d`
- [ ] Setup systemd service untuk auto-start
- [ ] Verifikasi aplikasi berjalan

**Dependencies:** Task 5.1  
**Estimasi:** 1 jam

### Task 5.3 — Monitoring Setup
- [ ] Setup log rotation (Docker logging driver)
- [ ] Setup health check monitoring (optional: UptimeRobot)
- [ ] Setup notifikasi error (optional: Telegram alert)

**Dependencies:** Task 5.2  
**Estimasi:** 30 menit

---

## Fase 6: Dokumentasi & Finalisasi

### Task 6.1 — Update Documentation
- [ ] Update `docs/SETUP.md` jika ada perubahan
- [ ] Update `agent/` files jika ada perubahan arsitektur
- [ ] Buat CHANGELOG.md

**Dependencies:** Semua implementasi  
**Estimasi:** 1 jam

### Task 6.2 — Code Review & Cleanup
- [ ] Review semua kode
- [ ] Hapus debug logs
- [ ] Pastikan tidak ada hardcoded values
- [ ] Pastikan semua error handling proper
- [ ] Pastikan semua context propagation benar

**Dependencies:** Semua implementasi  
**Estimasi:** 2 jam

---

## Timeline Estimasi

| Fase | Tasks | Estimasi Total |
|------|-------|:--------------:|
| Fase 1 | 3 tasks | 35 menit |
| Fase 2 | 7 tasks | 19 jam |
| Fase 3 | 2 tasks | 4 jam |
| Fase 4 | 2 tasks | 5 jam |
| Fase 5 | 3 tasks | 2.5 jam |
| Fase 6 | 2 tasks | 3 jam |
| **Total** | **19 tasks** | **~34 jam** |

---

## Prioritas Pengerjaan

### High Priority (Harus selesai dulu)
1. Task 1.1 — Go module init
2. Task 1.2 — Database migration
3. Task 2.1 — Config package
4. Task 2.2 — Database package
5. Task 2.3 — Telegram package
6. Task 2.5 — Rotation package
7. Task 2.6 — Scheduler package
8. Task 3.1 — Main application

### Medium Priority
9. Task 2.4 — Content/LLM package
10. Task 2.7 — Retry package
11. Task 3.2 — HTTP server
12. Task 4.1 — Integration test
13. Task 5.1 — Docker build

### Low Priority (Setelah semua jalan)
14. Task 4.2 — Manual testing
15. Task 5.2 — Deploy ke VPS
16. Task 5.3 — Monitoring
17. Task 6.1 — Documentation
18. Task 6.2 — Code review

---

## Dependencies Graph

```
Task 1.1 (Go Module)
  ├── Task 2.1 (Config)
  ├── Task 2.3 (Telegram)
  ├── Task 2.4 (Content/LLM)
  └── Task 2.7 (Retry)
         │
Task 1.2 (Database Migration)
  └── Task 2.2 (Database Repository)
         │
         ├── Task 2.5 (Rotation)
         └── Task 2.6 (Scheduler)
                │
                └── Task 3.1 (Main App)
                       │
                       ├── Task 3.2 (HTTP Server)
                       ├── Task 4.1 (Integration Test)
                       └── Task 5.1 (Docker Build)
                              │
                              └── Task 5.2 (Deploy VPS)
                                     │
                                     └── Task 5.3 (Monitoring)
```

---

## Definition of Done

Setiap task dianggap selesai jika:
- [ ] Kode sudah di-commit ke Git
- [ ] Unit test passing (`go test ./...`)
- [ ] Tidak ada linter errors
- [ ] Code review sudah dilakukan
- [ ] Dokumentasi sudah diupdate jika perlu