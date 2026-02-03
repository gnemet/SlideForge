-- SlideForge Initial Schema
-- PostgreSQL 18
CREATE TABLE pptx_files (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    original_file_path TEXT NOT NULL,
    template_file_path TEXT,
    thumbnail_dir_path TEXT,
    -- Path to stored PNGs
    metadata JSONB DEFAULT '{}',
    is_template BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE collected_slides (
    id SERIAL PRIMARY KEY,
    pptx_file_id INTEGER REFERENCES pptx_files(id),
    slide_number INTEGER NOT NULL,
    png_path TEXT NOT NULL,
    ai_analysis JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_pptx_metadata ON pptx_files USING GIN (metadata);
CREATE INDEX idx_slide_analysis ON collected_slides USING GIN (ai_analysis);