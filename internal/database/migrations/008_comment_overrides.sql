-- Migration to store user-modified comment text for PPTX slides.
-- This allows comments to act as editable placeholders.
CREATE TABLE IF NOT EXISTS comment_overrides (
    id SERIAL PRIMARY KEY,
    pptx_path TEXT NOT NULL,
    slide_number INTEGER NOT NULL,
    comment_index INTEGER NOT NULL,
    original_author TEXT,
    text TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pptx_path, slide_number, comment_index)
);
CREATE INDEX IF NOT EXISTS idx_comment_overrides_path ON comment_overrides(pptx_path);