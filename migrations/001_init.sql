-- Navisha Spark Database Schema
-- PostgreSQL (Supabase)
-- Version: 1.0

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- INSIGHTS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS insights (
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

-- Indexes for insights
CREATE INDEX IF NOT EXISTS idx_insights_category_level ON insights(category, level);
CREATE INDEX IF NOT EXISTS idx_insights_last_sent_at ON insights(last_sent_at);
CREATE INDEX IF NOT EXISTS idx_insights_tags ON insights USING GIN(tags);

-- ============================================
-- DELIVERY LOG TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS delivery_log (
    id SERIAL PRIMARY KEY,
    insight_id INT REFERENCES insights(id) ON DELETE CASCADE,
    sent_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success','failed','retry')),
    error_message TEXT,
    telegram_message_id BIGINT
);

-- Indexes for delivery_log
CREATE INDEX IF NOT EXISTS idx_delivery_log_sent_at ON delivery_log(sent_at);
CREATE INDEX IF NOT EXISTS idx_delivery_log_insight_id ON delivery_log(insight_id);
CREATE INDEX IF NOT EXISTS idx_delivery_log_status ON delivery_log(status);

-- ============================================
-- ROTATION STATE TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS rotation_state (
    category VARCHAR(100) PRIMARY KEY,
    last_sent_at TIMESTAMP,
    total_sent INT DEFAULT 0,
    last_level VARCHAR(20),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- SENT HISTORY TABLE (for deduplication)
-- ============================================
CREATE TABLE IF NOT EXISTS sent_history (
    insight_id INT REFERENCES insights(id) ON DELETE CASCADE,
    sent_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (insight_id, sent_at)
);

-- Index for deduplication queries
CREATE INDEX IF NOT EXISTS idx_sent_history_sent_at ON sent_history(sent_at);
CREATE INDEX IF NOT EXISTS idx_sent_history_insight_id ON sent_history(insight_id);

-- ============================================
-- TRIGGERS
-- ============================================

-- Update updated_at timestamp on insights
CREATE OR REPLACE FUNCTION update_insights_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_insights_updated_at') THEN
        CREATE TRIGGER trigger_insights_updated_at
            BEFORE UPDATE ON insights
            FOR EACH ROW
            EXECUTE FUNCTION update_insights_updated_at();
    END IF;
END $$;

-- Update updated_at timestamp on rotation_state
CREATE OR REPLACE FUNCTION update_rotation_state_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_rotation_state_updated_at') THEN
        CREATE TRIGGER trigger_rotation_state_updated_at
            BEFORE UPDATE ON rotation_state
            FOR EACH ROW
            EXECUTE FUNCTION update_rotation_state_updated_at();
    END IF;
END $$;

-- ============================================
-- CLEANUP FUNCTION (optional, run periodically)
-- ============================================
-- Remove old sent_history entries (older than 7 days)
-- This can be called via a cron job or manually
CREATE OR REPLACE FUNCTION cleanup_old_sent_history()
RETURNS void AS $$
BEGIN
    DELETE FROM sent_history
    WHERE sent_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- SAMPLE DATA (optional, for testing)
-- ============================================
-- Uncomment below to insert sample insights
-- Note: Use $$ quoting for text containing single quotes

-- INSERT INTO insights (category, level, title, insight, key, key_points, code_example, follow_ups, tags)
-- VALUES (
--     'Golang',
--     'beginner',
--     'Understanding Goroutine',
--     $$Goroutine adalah thread ringan yang dikelola oleh Go runtime. Goroutine memungkinkan eksekusi concurrent dengan biaya memory yang sangat kecil (mulai dari 2KB) dibandingkan thread OS (1MB).$$,
--     'goroutine-basics',
--     ARRAY['Goroutine lebih ringan dari thread OS', 'Dikelola oleh Go runtime', 'Mudah dibuat dengan keyword go'],
--     'go func() { fmt.Println("Hello") }()',
--     '[{"q":"Bagaimana cara membuat goroutine?", "a":"Gunakan keyword go sebelum fungsi: go myFunction()"}, {"q":"Apa perbedaan goroutine dan thread?", "a":"Goroutine lebih ringan, dikelola runtime, multiplexed ke thread OS"}]'::jsonb,
--     ARRAY['golang', 'concurrency', 'goroutine']
-- );

-- INSERT INTO insights (category, level, title, insight, key, key_points, code_example, follow_ups, tags)
-- VALUES (
--     'Database',
--     'intermediate',
--     'Transaction Isolation Levels',
--     $$Transaction isolation level menentukan seberapa satu transaksi terisolasi dari transaksi lain. PostgreSQL menyediakan 4 level: Read Uncommitted, Read Committed, Repeatable Read, dan Serializable.$$,
--     'transaction-isolation',
--     ARRAY['Read Committed adalah default di PostgreSQL', 'Serializable memberikan isolation tertinggi', 'Semakin tinggi isolation, semakin besar overhead'],
--     'SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;',
--     '[{"q":"Apa perbedaan Read Committed dan Repeatable Read?", "a":"Read Committed hanya melindungi dari dirty read, Repeatable Read melindungi dari non-repeatable read"}, {"q":"Apa itu dirty read?", "a":"Membaca data yang belum di-commit dari transaksi lain"}]'::jsonb,
--     ARRAY['database', 'postgresql', 'transaction', 'isolation']
-- );

-- INSERT INTO insights (category, level, title, insight, key, key_points, code_example, follow_ups, tags)
-- VALUES (
--     'System Design',
--     'advanced',
--     'CAP Theorem Deep Dive',
--     $$CAP Theorem menyatakan bahwa distributed system hanya bisa memenuhi 2 dari 3 properti: Consistency (semua node melihat data yang sama), Availability (setiap request mendapatkan response), Partition Tolerance (system tetap berjalan meskipun ada network partition).$$,
--     'cap-theorem',
--     ARRAY['Partition Tolerance adalah mandatory dalam distributed system', 'CP vs AP trade-off tergantung use case', 'Tidak ada sistem yang pure CAP di real world'],
--     '// CP System: etcd, ZooKeeper\n// AP System: Cassandra, DynamoDB',
--     '[{"q":"Bagaimana cara trade-off CAP?", "a":"Pilih CP jika konsistensi critical (financial), AP jika availability critical (social media)"}, {"q":"Apa yang dimaksud dengan eventual consistency?", "a":"Sistem yang akhirnya konsisten setelah partition healed, tapi tidak ada jaminan waktu"}]'::jsonb,
--     ARRAY['system-design', 'cap-theorem', 'distributed']
-- );

-- ============================================
-- ROTATION STATE (initial data for 13 categories)
-- ============================================
-- Uncomment below to insert initial rotation state

-- INSERT INTO rotation_state (category, last_sent_at, total_sent, last_level) VALUES
--     ('Golang', NULL, 0, NULL),
--     ('Data Structures & Algorithms', NULL, 0, NULL),
--     ('Coding Challenge', NULL, 0, NULL),
--     ('Database', NULL, 0, NULL),
--     ('System Design', NULL, 0, NULL),
--     ('API Design', NULL, 0, NULL),
--     ('Deployment / DevOps', NULL, 0, NULL),
--     ('Security', NULL, 0, NULL),
--     ('Network', NULL, 0, NULL),
--     ('Caching (Redis)', NULL, 0, NULL),
--     ('Message Broker (Kafka)', NULL, 0, NULL),
--     ('Distributed Systems', NULL, 0, NULL),
--     ('AI/ML untuk Backend Engineer', NULL, 0, NULL);
