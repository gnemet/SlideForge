-- Migration to add filename FTS
SET search_path TO slideforge,
    public;
ALTER TABLE pptx_files
ADD COLUMN IF NOT EXISTS fts_en tsvector GENERATED ALWAYS AS (to_tsvector('english', filename)) STORED;
ALTER TABLE pptx_files
ADD COLUMN IF NOT EXISTS fts_hu tsvector GENERATED ALWAYS AS (to_tsvector('hungarian', filename)) STORED;
ALTER TABLE pptx_files
ADD COLUMN IF NOT EXISTS fts_combined tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', filename), 'A') || setweight(to_tsvector('hungarian', filename), 'A')
    ) STORED;
CREATE INDEX IF NOT EXISTS idx_pptx_fts_en ON pptx_files USING GIN (fts_en);
CREATE INDEX IF NOT EXISTS idx_pptx_fts_hu ON pptx_files USING GIN (fts_hu);
CREATE INDEX IF NOT EXISTS idx_pptx_fts_combined ON pptx_files USING GIN (fts_combined);