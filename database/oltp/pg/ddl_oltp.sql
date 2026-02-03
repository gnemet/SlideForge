-- SlideForge OLTP Schema
-- PostgreSQL 18
-- Patterned after Jiramntr Ecosystem
-- Create and set schema
CREATE SCHEMA IF NOT EXISTS slideforge;
SET search_path TO slideforge,
    public;
-- 1. PPTX FILES
DROP TABLE IF EXISTS pptx_files CASCADE;
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
-- 2. COLLECTED SLIDES
DROP TABLE IF EXISTS collected_slides CASCADE;
CREATE TABLE collected_slides (
    id SERIAL PRIMARY KEY,
    pptx_file_id INTEGER REFERENCES pptx_files(id) ON DELETE CASCADE,
    slide_number INTEGER NOT NULL,
    png_path TEXT NOT NULL,
    ai_analysis JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- 3. INDEXES
CREATE INDEX IF NOT EXISTS idx_pptx_metadata ON pptx_files USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_slide_analysis ON collected_slides USING GIN (ai_analysis);
CREATE INDEX IF NOT EXISTS idx_pptx_filename ON pptx_files (filename);