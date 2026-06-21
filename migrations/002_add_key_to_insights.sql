-- Add key column to insights table for better prompt variation
-- Version: 2.0

-- Add key column
ALTER TABLE insights 
ADD COLUMN IF NOT EXISTS key VARCHAR(100);

-- Note: We allow multiple insights with the same (category, key) combination
-- This enables variation generation - when a key is reused, we create a variation
-- of an existing insight with that key instead of generating a completely new topic
-- The dedup window and content quality ensure we don't get repetitive content

-- Create index for faster key-based lookups
CREATE INDEX IF NOT EXISTS idx_insights_key 
ON insights(key) 
WHERE key IS NOT NULL;

-- Add comment to document the purpose of the key column
COMMENT ON COLUMN insights.key IS 'Topic/subtopic identifier used for prompt variation and deduplication. Format: category-specific slug (e.g., "goroutine-basics", "transaction-isolation").';