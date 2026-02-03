-- Migration to add Full-Text Search and Similarity capabilities
-- PostgreSQL 18
SET search_path TO slideforge,
    public;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
-- Add content and style_info to collected_slides if not exists
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'slideforge'
        AND table_name = 'collected_slides'
        AND column_name = 'content'
) THEN
ALTER TABLE collected_slides
ADD COLUMN content TEXT DEFAULT '';
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'slideforge'
        AND table_name = 'collected_slides'
        AND column_name = 'style_info'
) THEN
ALTER TABLE collected_slides
ADD COLUMN style_info JSONB DEFAULT '{}';
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'slideforge'
        AND table_name = 'collected_slides'
        AND column_name = 'fts_en'
) THEN
ALTER TABLE collected_slides
ADD COLUMN fts_en tsvector;
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'slideforge'
        AND table_name = 'collected_slides'
        AND column_name = 'fts_hu'
) THEN
ALTER TABLE collected_slides
ADD COLUMN fts_hu tsvector;
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'slideforge'
        AND table_name = 'collected_slides'
        AND column_name = 'fts_combined'
) THEN
ALTER TABLE collected_slides
ADD COLUMN fts_combined tsvector;
END IF;
END $$;
-- Search Settings Table
CREATE TABLE IF NOT EXISTS search_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO search_settings (key, value)
VALUES ('similarity_threshold', '0.3') ON CONFLICT (key) DO NOTHING;
INSERT INTO search_settings (key, value)
VALUES ('word_similarity_threshold', '0.3') ON CONFLICT (key) DO NOTHING;
-- Trigger function to update search vectors
CREATE OR REPLACE FUNCTION update_slide_search_vectors() RETURNS trigger AS $$ BEGIN NEW.fts_en := to_tsvector('english', coalesce(NEW.content, ''));
NEW.fts_hu := to_tsvector('hungarian', unaccent(coalesce(NEW.content, '')));
NEW.fts_combined := setweight(
    to_tsvector('english', coalesce(NEW.content, '')),
    'A'
) || setweight(
    to_tsvector('hungarian', unaccent(coalesce(NEW.content, ''))),
    'A'
);
RETURN NEW;
END $$ LANGUAGE plpgsql;
-- Create Trigger
DROP TRIGGER IF EXISTS trg_update_slide_search_vectors ON collected_slides;
CREATE TRIGGER trg_update_slide_search_vectors BEFORE
INSERT
    OR
UPDATE ON collected_slides FOR EACH ROW EXECUTE FUNCTION update_slide_search_vectors();
-- Indexes for FTS
CREATE INDEX IF NOT EXISTS idx_slides_fts_en ON collected_slides USING GIN (fts_en);
CREATE INDEX IF NOT EXISTS idx_slides_fts_hu ON collected_slides USING GIN (fts_hu);
CREATE INDEX IF NOT EXISTS idx_slides_fts_combined ON collected_slides USING GIN (fts_combined);
-- Index for similarity search
CREATE INDEX IF NOT EXISTS idx_slides_content_trgm ON collected_slides USING GIN (content gin_trgm_ops);